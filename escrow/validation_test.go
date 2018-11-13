package escrow

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/singnet/snet-daemon/blockchain"
)

func ChannelPaymentValidatorMock() *ChannelPaymentValidator {
	return &ChannelPaymentValidator{
		currentBlock:               func() (*big.Int, error) { return big.NewInt(99), nil },
		paymentExpirationThreshold: func() *big.Int { return big.NewInt(0) },
	}
}

type ValidationTestSuite struct {
	suite.Suite

	senderPrivateKey   *ecdsa.PrivateKey
	senderAddress      common.Address
	recipientAddress   common.Address
	mpeContractAddress common.Address

	validator ChannelPaymentValidator
}

func TestValidationTestSuite(t *testing.T) {
	suite.Run(t, new(ValidationTestSuite))
}

func (suite *ValidationTestSuite) SetupSuite() {
	suite.senderPrivateKey = generatePrivateKey()
	suite.senderAddress = crypto.PubkeyToAddress(suite.senderPrivateKey.PublicKey)
	suite.recipientAddress = crypto.PubkeyToAddress(generatePrivateKey().PublicKey)
	suite.mpeContractAddress = blockchain.HexToAddress("0xf25186b5081ff5ce73482ad761db0eb0d25abfbf")

	suite.validator = ChannelPaymentValidator{
		currentBlock:               func() (*big.Int, error) { return big.NewInt(99), nil },
		paymentExpirationThreshold: func() *big.Int { return big.NewInt(0) },
	}
}

func (suite *ValidationTestSuite) Payment() *Payment {
	payment := &Payment{
		Amount:             big.NewInt(12345),
		ChannelID:          big.NewInt(42),
		ChannelNonce:       big.NewInt(3),
		MpeContractAddress: suite.mpeContractAddress,
	}
	suite.Sign(payment, suite.senderPrivateKey)
	return payment
}

func (suite *ValidationTestSuite) Channel() *PaymentChannelData {
	return &PaymentChannelData{
		Nonce:            big.NewInt(3),
		Sender:           suite.senderAddress,
		Recipient:        suite.recipientAddress,
		GroupId:          big.NewInt(1),
		FullAmount:       big.NewInt(12345),
		Expiration:       big.NewInt(100),
		AuthorizedAmount: big.NewInt(12300),
		Signature:        nil,
	}
}

func (suite *ValidationTestSuite) Sign(payment *Payment, privateKey *ecdsa.PrivateKey) {
	message := bytes.Join([][]byte{
		payment.MpeContractAddress.Bytes(),
		bigIntToBytes(payment.ChannelID),
		bigIntToBytes(payment.ChannelNonce),
		bigIntToBytes(payment.Amount),
	}, nil)

	payment.Signature = getSignature(message, privateKey)
}

func (suite *ValidationTestSuite) TestPaymentIsValid() {
	payment := suite.Payment()
	channel := suite.Channel()

	err := suite.validator.Validate(payment, channel)

	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
}

func (suite *ValidationTestSuite) TestValidatePaymentChannelNonce() {
	payment := suite.Payment()
	payment.ChannelNonce = big.NewInt(2)
	suite.Sign(payment, suite.senderPrivateKey)
	channel := suite.Channel()
	channel.Nonce = big.NewInt(3)

	err := suite.validator.Validate(payment, channel)

	assert.Equal(suite.T(), NewPaymentError(Unauthenticated, "incorrect payment channel nonce, latest: 3, sent: 2"), err)
}

func (suite *ValidationTestSuite) TestValidatePaymentIncorrectSignatureLength() {
	payment := suite.Payment()
	payment.Signature = blockchain.HexToBytes("0x0000")

	err := suite.validator.Validate(payment, suite.Channel())

	assert.Equal(suite.T(), NewPaymentError(Unauthenticated, "payment signature is not valid"), err)
}

func (suite *ValidationTestSuite) TestValidatePaymentIncorrectSignatureChecksum() {
	payment := suite.Payment()
	payment.Signature = blockchain.HexToBytes("0xa4d2ae6f3edd1f7fe77e4f6f78ba18d62e6093bcae01ef86d5de902d33662fa372011287ea2d8d8436d9db8a366f43480678df25453b484c67f80941ef2c05ef21")

	err := suite.validator.Validate(payment, suite.Channel())

	assert.Equal(suite.T(), NewPaymentError(Unauthenticated, "payment signature is not valid"), err)
}

func (suite *ValidationTestSuite) TestValidatePaymentIncorrectSigner() {
	payment := suite.Payment()
	payment.Signature = blockchain.HexToBytes("0xa4d2ae6f3edd1f7fe77e4f6f78ba18d62e6093bcae01ef86d5de902d33662fa372011287ea2d8d8436d9db8a366f43480678df25453b484c67f80941ef2c05ef01")

	err := suite.validator.Validate(payment, suite.Channel())

	assert.Equal(suite.T(), NewPaymentError(Unauthenticated, "payment is not signed by channel sender"), err)
}

func (suite *ValidationTestSuite) TestValidatePaymentChannelCannotGetCurrentBlock() {
	validator := &ChannelPaymentValidator{
		currentBlock: func() (*big.Int, error) { return nil, errors.New("blockchain error") },
	}

	err := validator.Validate(suite.Payment(), suite.Channel())

	assert.Equal(suite.T(), NewPaymentError(Internal, "cannot determine current block"), err)
}

func (suite *ValidationTestSuite) TestValidatePaymentExpiredChannel() {
	validator := &ChannelPaymentValidator{
		currentBlock:               func() (*big.Int, error) { return big.NewInt(99), nil },
		paymentExpirationThreshold: func() *big.Int { return big.NewInt(0) },
	}
	channel := suite.Channel()
	channel.Expiration = big.NewInt(99)

	err := validator.Validate(suite.Payment(), channel)

	assert.Equal(suite.T(), NewPaymentError(Unauthenticated, "payment channel is near to be expired, expiration time: 99, current block: 99, expiration threshold: 0"), err)
}

func (suite *ValidationTestSuite) TestValidatePaymentChannelExpirationThreshold() {
	validator := &ChannelPaymentValidator{
		currentBlock:               func() (*big.Int, error) { return big.NewInt(98), nil },
		paymentExpirationThreshold: func() *big.Int { return big.NewInt(1) },
	}
	channel := suite.Channel()
	channel.Expiration = big.NewInt(99)

	err := validator.Validate(suite.Payment(), channel)

	assert.Equal(suite.T(), NewPaymentError(Unauthenticated, "payment channel is near to be expired, expiration time: 99, current block: 98, expiration threshold: 1"), err)
}

func (suite *ValidationTestSuite) TestValidatePaymentAmountIsTooBig() {
	payment := suite.Payment()
	payment.Amount = big.NewInt(12346)
	suite.Sign(payment, suite.senderPrivateKey)
	channel := suite.Channel()
	channel.FullAmount = big.NewInt(12345)

	err := suite.validator.Validate(payment, suite.Channel())

	assert.Equal(suite.T(), NewPaymentError(Unauthenticated, "not enough tokens on payment channel, channel amount: 12345, payment amount: 12346"), err)
}
