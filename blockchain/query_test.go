package blockchain
import (
	"testing"

	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
)
func TestE2E(t *testing.T) {
	t.Run("Base", func(t *testing.T) {
		configPath := "../config_e2e.yaml"
		Run(t, config.FromFile(configPath))
	})

	// t.Run("NoOrderer", func(t *testing.T) {
	// 	//Using setup done set above by end to end test, run below test with new config which has no orderer config inside
	// 	runWithNoOrdererConfig(t, config.FromFile(integration.GetConfigPath("config_e2e_no_orderer.yaml")))
	// })
}