package basesuite

import (
	"context"

	"cosmossdk.io/math"

	"github.com/bnb-chain/greenfield-sp-standard-test/config"
	"github.com/bnb-chain/greenfield-sp-standard-test/core/log"
	types2 "github.com/bnb-chain/greenfield/sdk/types"
	spTypes "github.com/bnb-chain/greenfield/x/sp/types"
)

type BaseSuite struct {
	RootAcc     *Account
	TestAcc     *Account
	SPInfo      *spTypes.StorageProvider
	TestResults []Output
}

func (s *BaseSuite) SetupSuite() {
	s.RootAcc = NewAccount(config.CfgEnv.GreenfieldEndpoint, config.CfgEnv.GreenfieldChainId, config.CfgEnv.SPAddr, config.CfgEnv.RootAcc)
	rootAccountBalance, err := s.RootAcc.SDKClient.GetAccountBalance(context.Background(), s.RootAcc.Addr.String())
	if err != nil {
		log.Errorf("Error getting account balance: %v", err)
	}
	log.Infof("rootAccountBalance: %v , root account : %s", rootAccountBalance.Amount, s.RootAcc.Addr.String())
	if rootAccountBalance.Amount.LT(math.NewInt(1e18)) {
		log.Fatalf("rootAccount balance less 1BNB, need more BNB balance for test")
	}

	s.TestAcc = NewAccount(config.CfgEnv.GreenfieldEndpoint, config.CfgEnv.GreenfieldChainId, config.CfgEnv.SPAddr, config.CfgEnv.TestAcc)
	// check test account balance
	s.InitAccountsBNBBalance([]*Account{s.TestAcc}, 3e17)

	spAddr := config.CfgEnv.SPAddr
	spInfo, _ := s.TestAcc.SelectSP(spAddr)
	s.SPInfo = spInfo
	log.Infof("SP Endpoint: %s, address: %s", spInfo.Endpoint, spInfo.OperatorAddress)

}
func (s *BaseSuite) InitAccountsBNBBalance(accounts []*Account, amount int64) {
	for _, normalAccount := range accounts {
		accountBalance, err := normalAccount.SDKClient.GetAccountBalance(context.Background(), normalAccount.Addr.String())
		if err != nil {
			log.Errorf("Error getting account balance: %v", err)
		}
		if accountBalance.Amount.LT(math.NewInt(amount)) {
			transferTxHash, err := s.RootAcc.SDKClient.Transfer(context.Background(), normalAccount.Addr.String(), math.NewInt(amount), types2.TxOption{})
			if err != nil {
				log.Errorf("root account Transfer err: %v", err)
			}
			log.Infof("Transfer BNB to: %s, tx hash: %v", normalAccount.Addr.String(), transferTxHash)
			txInfo, err := s.RootAcc.SDKClient.WaitForTx(context.Background(), transferTxHash)
			if err != nil || txInfo.Code != 0 {
				log.Errorf("root account Transfer WaitForTx err: %v", err)
			}
		}
	}
}
