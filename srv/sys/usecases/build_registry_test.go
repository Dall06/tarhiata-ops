package usecases

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestBuildRegistry_NewJob_UniqueIDs(t *testing.T) {
	reg := NewBuildRegistry()
	j1 := reg.NewJob("svc-a")
	j2 := reg.NewJob("svc-b")

	if j1.ID == "" || j2.ID == "" {
		t.Fatal("se esperaban IDs no vacíos")
	}
	if j1.ID == j2.ID {
		t.Error("se esperaban IDs distintos para builds distintos")
	}
	if reg.Get(j1.ID) != j1 || reg.Get(j2.ID) != j2 {
		t.Error("Get no devolvió la instancia correcta por ID")
	}
	if reg.Get("no-existe") != nil {
		t.Error("se esperaba nil para un ID inexistente")
	}
}

// TestBuildJob_SubscribeReceivesAccumulatedAndLiveLines valida el flujo completo: un
// suscriptor que se engancha después de que ya corrieron algunas líneas debe recibir esas
// líneas acumuladas PRIMERO, y luego las nuevas en vivo, hasta que el job termina.
func TestBuildJob_SubscribeReceivesAccumulatedAndLiveLines(t *testing.T) {
	reg := NewBuildRegistry()
	job := reg.NewJob("web-api")

	job.AppendLine("clonando...")
	job.AppendLine("construyendo...")

	soFar, live := job.Subscribe()
	if len(soFar) != 2 || soFar[0] != "clonando..." || soFar[1] != "construyendo..." {
		t.Fatalf("líneas acumuladas inesperadas: %v", soFar)
	}
	if live == nil {
		t.Fatal("se esperaba un canal vivo porque el job no ha terminado")
	}

	go func() {
		job.AppendLine("publicando...")
		job.Finish("success", "tarhiata-registry:5000/web-api:abc123", "")
	}()

	var received []string
	for line := range live {
		received = append(received, line)
	}
	if len(received) != 1 || received[0] != "publicando..." {
		t.Errorf("se esperaba recibir la línea en vivo, got %v", received)
	}
	if job.Status != "success" || job.ImageTag != "tarhiata-registry:5000/web-api:abc123" {
		t.Errorf("estado final inesperado: status=%s imageTag=%s", job.Status, job.ImageTag)
	}
}

func TestBuildJob_SubscribeAfterFinish_NoLiveChannel(t *testing.T) {
	job := &BuildJob{ID: "x", Service: "svc"}
	job.AppendLine("ya terminado")
	job.Finish("failed", "", "boom")

	soFar, live := job.Subscribe()
	if len(soFar) != 1 || soFar[0] != "ya terminado" {
		t.Fatalf("líneas acumuladas inesperadas: %v", soFar)
	}
	if live != nil {
		t.Error("no se esperaba un canal vivo para un job ya terminado")
	}
	if !job.IsDone() {
		t.Error("se esperaba IsDone()=true")
	}
}

// TestBuildRegistry_Concurrency valida con -race que builds concurrentes de distintos
// servicios no se pisan entre sí (cada uno con su propio ID, lines y suscriptores).
func TestBuildRegistry_Concurrency(t *testing.T) {
	reg := NewBuildRegistry()
	var wg sync.WaitGroup

	for i := range 20 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			job := reg.NewJob(fmt.Sprintf("svc-%d", n))
			for l := range 5 {
				job.AppendLine(fmt.Sprintf("line-%d", l))
			}
			_, live := job.Subscribe()
			job.Finish("success", "tag", "")
			if live != nil {
				for range live {
				}
			}
		}(i)
	}
	wg.Wait()

	if len(reg.jobs) != 20 {
		t.Errorf("se esperaban 20 jobs registrados, got %d", len(reg.jobs))
	}
}

func TestBuildJob_SlowSubscriberDoesNotBlockBuild(t *testing.T) {
	job := &BuildJob{ID: "x", Service: "svc"}
	_, live := job.Subscribe()

	done := make(chan struct{})
	go func() {
		// Nunca se drena "live" a propósito: AppendLine no debe bloquearse por esto
		// gracias al buffer + select/default.
		for i := range 100 {
			job.AppendLine(fmt.Sprintf("line-%d", i))
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("AppendLine se bloqueó por un suscriptor lento que nunca lee")
	}
	_ = live
}
