package components

import (
	"crypto/rand"
	"math/big"
)

// SecretTargetComponent modela o SigiloMetaComponent responsável pela geração protegida de metas
type SecretTargetComponent struct {
	min int
	max int
}

// NewSecretTargetComponent inicializa o componente com limites [5, 95]
func NewSecretTargetComponent() *SecretTargetComponent {
	return &SecretTargetComponent{
		min: 5,
		max: 95,
	}
}

// Generate sorteia de forma criptograficamente segura uma meta no intervalo [min, max]
func (c *SecretTargetComponent) Generate() int {
	delta := int64(c.max - c.min + 1)
	n, err := rand.Int(rand.Reader, big.NewInt(delta))
	if err != nil {
		// Fallback seguro caso haja falha no CSPRNG
		return 50
	}
	return int(n.Int64()) + c.min
}
