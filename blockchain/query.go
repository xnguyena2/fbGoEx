package blockchain

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/orderer"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	mspclient "github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/resource"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/stretchr/testify/assert"
)

func queryCC(t *testing.T, client *channel.Client, targetEndpoints ...string) []byte {
	response, err := client.Query(channel.Request{ChaincodeID: "mycc", Fcn: "query", Args: [][]byte{[]byte("a")}},
		channel.WithRetry(retry.DefaultChannelOpts),
		channel.WithTargetEndpoints(targetEndpoints...),
	)
	if err != nil {
		t.Fatalf("Failed to query funds: %s", err)
	}
	return response.Payload
}

func executeCC(t *testing.T, client *channel.Client) {
	_, err := client.Execute(channel.Request{ChaincodeID: "mycc", Fcn: "open", Args: [][]byte{[]byte("a"), []byte("33")}},
		channel.WithRetry(retry.DefaultChannelOpts))
	if err != nil {
		t.Fatalf("Failed to move funds: %s", err)
	}
}

// ModifyMaxMessageCount increments the orderer's BatchSize.MaxMessageCount in a channel config
func ModifyMaxMessageCount(config *common.Config) (uint32, error) {

	// Modify Config
	batchSizeBytes := config.ChannelGroup.Groups["Orderer"].Values["BatchSize"].Value
	batchSize := &orderer.BatchSize{}
	if err := proto.Unmarshal(batchSizeBytes, batchSize); err != nil {
		return 0, err
	}
	batchSize.MaxMessageCount = batchSize.MaxMessageCount + 1
	newMatchSizeBytes, err := proto.Marshal(batchSize)
	if err != nil {
		return 0, err
	}
	config.ChannelGroup.Groups["Orderer"].Values["BatchSize"].Value = newMatchSizeBytes

	return batchSize.MaxMessageCount, nil
}

func fetchConfigBlock(t *testing.T, client *resmgmt.Client, adminOrg msp.SigningIdentity) {
	currentConfigBlock, err := client.QueryConfigBlockFromOrderer("blockchainchannel", resmgmt.WithOrdererEndpoint("orderer.example.com"))
	if err != nil {
		t.Fatalf("Failed to fetch config block: %s", err)
	}
	//fmt.Println(currentConfigBlock)
	originalConfig, err := resource.ExtractConfigFromBlock(currentConfigBlock)
	assert.Nil(t, err, "extractConfigFromBlock failed")

	// Prepare new configuration
	modifiedConfigBytes, err := proto.Marshal(originalConfig)
	assert.Nil(t, err, "error marshalling originalConfig")
	modifiedConfig := &common.Config{}
	assert.Nil(t, proto.Unmarshal(modifiedConfigBytes, modifiedConfig), "unmarshalling modifiedConfig failed")
	newMaxMessageCount, err := ModifyMaxMessageCount(modifiedConfig)
	fmt.Println(newMaxMessageCount)
	assert.Nil(t, err, "error modifying config")
	configUpdate, err := resmgmt.CalculateConfigUpdate("blockchainchannel", originalConfig, modifiedConfig)

	configUpdateEnv, err := resource.CreateConfigUpdateEnv(configUpdate, nil)
	if err != nil {
		t.Fatalf("Failed to fetch config block: %s", err)
	}
	configUpdateBytes, err := proto.Marshal(configUpdateEnv)
	r := bytes.NewReader(configUpdateBytes)
	req := resmgmt.SaveChannelRequest{ChannelID: "blockchainchannel",
		ChannelConfig:     r,
		SigningIdentities: []msp.SigningIdentity{adminOrg}}
	txID, err := client.SaveChannel(req, resmgmt.WithRetry(retry.DefaultResMgmtOpts), resmgmt.WithOrdererEndpoint("orderer.example.com"))
	if err != nil {
		t.Fatalf("Failed to fetch config block: %s", err)
	}
	fmt.Println(txID)
}

//Hello sample
func Hello() {
	fmt.Println("Hello from Blockchain!")
}

// Run enables testing an end-to-end scenario against the supplied SDK options
func Run(t *testing.T, configOpt core.ConfigProvider, sdkOpts ...fabsdk.Option) {

	sdk, err := fabsdk.New(configOpt, sdkOpts...)
	if err != nil {
		t.Fatalf("Failed to create new SDK: %s", err)
	}
	defer sdk.Close()

	//prepare channel client context using client context
	//clientChannelContext := sdk.ChannelContext("blockchainchannel", fabsdk.WithUser("User1"), fabsdk.WithOrg("Org1"))

	clientContext := sdk.Context(fabsdk.WithUser("Admin"), fabsdk.WithOrg("Org1"))
	oordererMspClient, err := mspclient.New(sdk.Context(), mspclient.WithOrg("OrdererOrg"))
	if err != nil {
		t.Fatalf("Failed to create msp org1: %s", err)
	}
	ordererAdminUser, err := oordererMspClient.GetSigningIdentity("Admin")
	resMgmtClient, err := resmgmt.New(clientContext)
	if err != nil {
		t.Fatalf("Failed to create channel management client: %s", err)
	}
	fetchConfigBlock(t, resMgmtClient, ordererAdminUser)

	/*
		// Channel client is used to query and execute transactions (Org1 is default org)
		client, err := channel.New(clientChannelContext)
		if err != nil {
			t.Fatalf("Failed to create new channel client: %s", err)
		}

		executeCC(t, client)

		result := string(queryCC(t, client))
		fmt.Println(result)
	*/
}
