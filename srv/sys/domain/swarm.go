package domain

// SwarmServiceInfo encapsula los metadatos de un servicio orquestado en Docker Swarm.
type SwarmServiceInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Mode     string `json:"mode"`
	Replicas string `json:"replicas"`
	Image    string `json:"image"`
	Ports    string `json:"ports"`
	Domain   string `json:"domain"`
	Expose   bool   `json:"expose"`
}

// SwarmDBInfo encapsula el estado y conexión de una base de datos registrada en el cluster.
type SwarmDBInfo struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Engine         string `json:"engine"`
	DeployType     string `json:"deployType"` // "single-node", "multi-node", "external"
	InternalPort   int    `json:"internalPort"`
	ExternalURL    string `json:"externalUrl"`
	TargetNode     string `json:"targetNode"`
	Status         string `json:"status"` // "running", "external", "offline"
	InternalDNS    string `json:"internalDns"`
}

// SwarmNodeInfo representa un nodo miembro del clúster Swarm (Manager o Worker).
type SwarmNodeInfo struct {
	ID            string `json:"id"`
	Hostname      string `json:"hostname"`
	Status        string `json:"status"`
	Availability  string `json:"availability"`
	ManagerStatus string `json:"managerStatus"`
	EngineVersion string `json:"engineVersion"`
}

// SwarmStatus consolida la salud, servicios, bases de datos, nodos y accesos web del clúster Swarm.
type SwarmStatus struct {
	Active        bool               `json:"active"`
	NodeState     string             `json:"nodeState"`
	DockerVersion string             `json:"dockerVersion"`
	Services      []SwarmServiceInfo `json:"services"`
	Databases     []SwarmDBInfo      `json:"databases"`
	Nodes         []SwarmNodeInfo    `json:"nodes"`
	Dashboards    map[string]string  `json:"dashboards"`
}

