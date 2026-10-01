package usecases

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// BuildJob representa un build en curso o terminado, disparado por un webhook (donde
// nadie espera sincrónicamente el resultado) para que el dashboard pueda "engancharse"
// a verlo mientras corre, vía streaming.
type BuildJob struct {
	ID        string
	Service   string
	Status    string // "running" | "success" | "failed"
	ImageTag  string
	Error     string
	StartedAt time.Time

	mu    sync.Mutex
	lines []string
	subs  []chan string
	done  bool
}

// AppendLine agrega una línea de log y la transmite a cualquier suscriptor activo.
func (j *BuildJob) AppendLine(line string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.lines = append(j.lines, line)
	for _, ch := range j.subs {
		select {
		case ch <- line:
		default: // suscriptor lento: no bloquear el build por un consumidor atascado
		}
	}
}

// Finish marca el job como terminado (éxito o error) y cierra los canales de los
// suscriptores activos.
func (j *BuildJob) Finish(status, imageTag, errMsg string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = status
	j.ImageTag = imageTag
	j.Error = errMsg
	j.done = true
	for _, ch := range j.subs {
		close(ch)
	}
	j.subs = nil
}

// Subscribe devuelve las líneas ya acumuladas y, si el job sigue corriendo, un canal
// que recibe las líneas nuevas en vivo (nil si ya terminó: no hay nada más que esperar).
func (j *BuildJob) Subscribe() (linesSoFar []string, live <-chan string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	linesSoFar = append([]string(nil), j.lines...)
	if j.done {
		return linesSoFar, nil
	}
	ch := make(chan string, 64)
	j.subs = append(j.subs, ch)
	return linesSoFar, ch
}

// IsDone reporta si el job ya terminó (para que el caller del stream sepa cuándo cerrar).
func (j *BuildJob) IsDone() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.done
}

// BuildRegistry administra los BuildJob activos en memoria, igual de simple que
// sshclient.Pool: vive y muere con el proceso, no se persiste a SQLite.
type BuildRegistry struct {
	mu   sync.Mutex
	jobs map[string]*BuildJob
}

func NewBuildRegistry() *BuildRegistry {
	return &BuildRegistry{jobs: make(map[string]*BuildJob)}
}

// NewJob crea y registra un BuildJob nuevo con un ID aleatorio.
func (r *BuildRegistry) NewJob(service string) *BuildJob {
	job := &BuildJob{
		ID:        generateBuildID(),
		Service:   service,
		Status:    "running",
		StartedAt: time.Now(),
	}
	r.mu.Lock()
	r.jobs[job.ID] = job
	r.mu.Unlock()
	return job
}

// Get busca un job por ID. Devuelve nil si no existe (ej. ya se perdió al reiniciar el proceso).
func (r *BuildRegistry) Get(id string) *BuildJob {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.jobs[id]
}

func generateBuildID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))[:16]
	}
	return hex.EncodeToString(b)
}
