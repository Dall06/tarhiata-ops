package ports

// Provisioner define el contrato para herramientas de IaC (Terraform)
type Provisioner interface {
	// ProvisionNode crea o actualiza un nodo (Droplet/EC2) y retorna su IP Pública y la llave privada SSH generada
	ProvisionNode(token string, nodeName string, region string) (string, string, error)
	
	// DestroyNode destruye la infraestructura de un nodo por su nombre
	DestroyNode(token string, nodeName string) error
}
