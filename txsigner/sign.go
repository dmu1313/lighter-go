package txsigner

import (
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/dmu1313/lighter-go/client"
	"github.com/dmu1313/lighter-go/client/http"
	"github.com/dmu1313/lighter-go/types"

	"github.com/dmu1313/lighter-go/types/txtypes"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

var chainId uint32

type SignedTxResponse struct {
	TxType        uint8
	TxInfo        string
	TxHash        string
	MessageToSign string
	Err           error
}

type APIKeyResponse struct {
	PrivateKey string
	PublicKey  string
	Err        error
}

func messageToSign(txInfo txtypes.TxInfo) string {
	switch typed := txInfo.(type) {
	case *txtypes.L2ChangePubKeyTxInfo:
		return typed.GetL1SignatureBody()
	case *txtypes.L2TransferTxInfo:
		return typed.GetL1SignatureBody(chainId)
	default:
		return ""
	}
}

func signedTxResponseErr(err error) SignedTxResponse {
	return SignedTxResponse{Err: err}
}

func signedTxResponsePanic(err error) SignedTxResponse {
	return signedTxResponseErr(fmt.Errorf("panic: %v", err))
}

func convertTxInfoToResponse(txInfo txtypes.TxInfo, err error) SignedTxResponse {
	if err != nil {
		return signedTxResponseErr(err)
	}
	if txInfo == nil {
		return signedTxResponseErr(errors.New("nil transaction info"))
	}

	txInfoStr, err := txInfo.GetTxInfo()
	if err != nil {
		return signedTxResponseErr(err)
	}

	resp := SignedTxResponse{
		TxType: uint8(txInfo.GetTxType()),
		TxInfo: string(txInfoStr),
		TxHash: string(txInfo.GetTxHash()),
	}

	if msg := messageToSign(txInfo); msg != "" {
		resp.MessageToSign = msg
	}

	return resp
}

// getClient returns the go TxClient from the specified cApiKeyIndex and cAccountIndex
func getClient(cApiKeyIndex int32, cAccountIndex int64) (*client.TxClient, error) {
	apiKeyIndex := uint8(cApiKeyIndex)
	accountIndex := int64(cAccountIndex)
	return client.GetClient(apiKeyIndex, accountIndex)
}

func getTransactOpts(cNonce int64) *types.TransactOpts {
	nonce := int64(cNonce)
	return &types.TransactOpts{
		Nonce: &nonce,
	}
}

//export GenerateAPIKey
func GenerateAPIKey(cSeed string) (ret APIKeyResponse) {
	defer func() {
		if r := recover(); r != nil {
			ret = APIKeyResponse{Err: fmt.Errorf("panic: %v", r)}
		}
	}()

	seed := cSeed
	privateKeyStr, publicKeyStr, err := client.GenerateAPIKey(seed)
	if err != nil {
		return APIKeyResponse{Err: err}
	}

	return APIKeyResponse{
		PrivateKey: privateKeyStr,
		PublicKey:  publicKeyStr,
	}
}

//export CreateClient
func CreateClient(cUrl string, cPrivateKey string, cChainId int32, cApiKeyIndex int32, cAccountIndex int64) (ret error) {
	defer func() {
		if r := recover(); r != nil {
			ret = fmt.Errorf("panic: %v", r)
		}
	}()

	url := cUrl
	privateKey := cPrivateKey
	chainId = uint32(cChainId)
	apiKeyIndex := uint8(cApiKeyIndex)
	accountIndex := int64(cAccountIndex)

	httpClient := http.NewClient(url)

	_, err := client.CreateClient(httpClient, privateKey, chainId, apiKeyIndex, accountIndex)
	return err
}

//export CheckClient
func CheckClient(cApiKeyIndex int32, cAccountIndex int64) (ret error) {
	defer func() {
		if r := recover(); r != nil {
			ret = fmt.Errorf("panic: %v", r)
		}
	}()

	c, err := getClient(cApiKeyIndex, cAccountIndex)
	if err != nil {
		return err
	}

	return c.Check()
}

//export SignChangePubKey
func SignChangePubKey(cPubKey string, cNonce int64, cApiKeyIndex int32, cAccountIndex int64) (ret SignedTxResponse) {
	defer func() {
		if r := recover(); r != nil {
			ret = signedTxResponsePanic(fmt.Errorf("%v", r))
		}
	}()

	c, err := getClient(cApiKeyIndex, cAccountIndex)
	if err != nil {
		return signedTxResponseErr(err)
	}

	pubKeyStr := cPubKey
	pubKeyBytes, err := hexutil.Decode(pubKeyStr)
	if err != nil {
		return signedTxResponseErr(err)
	}
	if len(pubKeyBytes) != 40 {
		return signedTxResponseErr(fmt.Errorf("invalid pub key length. expected 40 but got %v", len(pubKeyBytes)))
	}
	var pubKey [40]byte
	copy(pubKey[:], pubKeyBytes)

	tx := &types.ChangePubKeyReq{
		PubKey: pubKey,
	}
	ops := getTransactOpts(cNonce)

	txInfo, err := c.GetChangePubKeyTransaction(tx, ops)
	return convertTxInfoToResponse(txInfo, err)
}

//export SignCreateOrder
func SignCreateOrder(cMarketIndex int32, cClientOrderIndex int64, cBaseAmount int64, cPrice int32, cIsAsk int32, cOrderType int32, cTimeInForce int32, cReduceOnly int32, cTriggerPrice int32, cOrderExpiry int64, cNonce int64, cApiKeyIndex int32, cAccountIndex int64) (ret SignedTxResponse) {
	defer func() {
		if r := recover(); r != nil {
			ret = signedTxResponsePanic(fmt.Errorf("%v", r))
		}
	}()

	c, err := getClient(cApiKeyIndex, cAccountIndex)
	if err != nil {
		return signedTxResponseErr(err)
	}

	marketIndex := int16(cMarketIndex)
	clientOrderIndex := int64(cClientOrderIndex)
	baseAmount := int64(cBaseAmount)
	price := uint32(cPrice)
	isAsk := uint8(cIsAsk)
	orderType := uint8(cOrderType)
	timeInForce := uint8(cTimeInForce)
	reduceOnly := uint8(cReduceOnly)
	triggerPrice := uint32(cTriggerPrice)
	orderExpiry := int64(cOrderExpiry)

	if orderExpiry == -1 {
		orderExpiry = time.Now().Add(time.Hour * 24 * 28).UnixMilli() // 28 days
	}

	tx := &types.CreateOrderTxReq{
		MarketIndex:      marketIndex,
		ClientOrderIndex: clientOrderIndex,
		BaseAmount:       baseAmount,
		Price:            price,
		IsAsk:            isAsk,
		Type:             orderType,
		TimeInForce:      timeInForce,
		ReduceOnly:       reduceOnly,
		TriggerPrice:     triggerPrice,
		OrderExpiry:      orderExpiry,
	}
	ops := getTransactOpts(cNonce)

	txInfo, err := c.GetCreateOrderTransaction(tx, ops)
	return convertTxInfoToResponse(txInfo, err)
}

//export SignCreateGroupedOrders
func SignCreateGroupedOrders(cGroupingType uint8, cOrders []types.CreateOrderTxReq, cLen int32, cNonce int64, cApiKeyIndex int32, cAccountIndex int64) (ret SignedTxResponse) {
	defer func() {
		if r := recover(); r != nil {
			ret = signedTxResponsePanic(fmt.Errorf("%v", r))
		}
	}()

	c, err := getClient(cApiKeyIndex, cAccountIndex)
	if err != nil {
		return signedTxResponseErr(err)
	}

	length := int(cLen)
	orders := make([]*types.CreateOrderTxReq, length)
	// size := unsafe.Sizeof(*cOrders)

	for i := 0; i < length; i++ {
		order := cOrders[i]
		// order := (*C.CreateOrderTxReq)(unsafe.Pointer(uintptr(unsafe.Pointer(cOrders)) + uintptr(i)*uintptr(size)))

		orderExpiry := int64(order.OrderExpiry)
		if orderExpiry == -1 {
			orderExpiry = time.Now().Add(time.Hour * 24 * 28).UnixMilli()
		}

		orders[i] = &types.CreateOrderTxReq{
			MarketIndex:      int16(order.MarketIndex),
			ClientOrderIndex: int64(order.ClientOrderIndex),
			BaseAmount:       int64(order.BaseAmount),
			Price:            uint32(order.Price),
			IsAsk:            uint8(order.IsAsk),
			Type:             uint8(order.Type),
			TimeInForce:      uint8(order.TimeInForce),
			ReduceOnly:       uint8(order.ReduceOnly),
			TriggerPrice:     uint32(order.TriggerPrice),
			OrderExpiry:      orderExpiry,
		}
	}

	tx := &types.CreateGroupedOrdersTxReq{
		GroupingType: uint8(cGroupingType),
		Orders:       orders,
	}
	ops := getTransactOpts(cNonce)

	txInfo, err := c.GetCreateGroupedOrdersTransaction(tx, ops)
	return convertTxInfoToResponse(txInfo, err)
}

//export SignCancelOrder
func SignCancelOrder(cMarketIndex int32, cOrderIndex int64, cNonce int64, cApiKeyIndex int32, cAccountIndex int64) (ret SignedTxResponse) {
	defer func() {
		if r := recover(); r != nil {
			ret = signedTxResponsePanic(fmt.Errorf("%v", r))
		}
	}()

	c, err := getClient(cApiKeyIndex, cAccountIndex)
	if err != nil {
		return signedTxResponseErr(err)
	}

	marketIndex := int16(cMarketIndex)
	orderIndex := int64(cOrderIndex)

	tx := &types.CancelOrderTxReq{
		MarketIndex: marketIndex,
		Index:       orderIndex,
	}
	ops := getTransactOpts(cNonce)

	txInfo, err := c.GetCancelOrderTransaction(tx, ops)
	return convertTxInfoToResponse(txInfo, err)
}

//export SignWithdraw
func SignWithdraw(cAssetIndex int32, cRouteType int32, cAmount uint64, cNonce int64, cApiKeyIndex int32, cAccountIndex int64) (ret SignedTxResponse) {
	defer func() {
		if r := recover(); r != nil {
			ret = signedTxResponsePanic(fmt.Errorf("%v", r))
		}
	}()

	c, err := getClient(cApiKeyIndex, cAccountIndex)
	if err != nil {
		return signedTxResponseErr(err)
	}

	assetIndex := int16(cAssetIndex)
	routeType := uint8(cRouteType)
	amount := uint64(cAmount)

	tx := &types.WithdrawTxReq{
		AssetIndex: assetIndex,
		RouteType:  routeType,
		Amount:     amount,
	}
	ops := getTransactOpts(cNonce)

	txInfo, err := c.GetWithdrawTransaction(tx, ops)
	return convertTxInfoToResponse(txInfo, err)
}

//export SignCreateSubAccount
func SignCreateSubAccount(cNonce int64, cApiKeyIndex int32, cAccountIndex int64) (ret SignedTxResponse) {
	defer func() {
		if r := recover(); r != nil {
			ret = signedTxResponsePanic(fmt.Errorf("%v", r))
		}
	}()

	c, err := getClient(cApiKeyIndex, cAccountIndex)
	if err != nil {
		return signedTxResponseErr(err)
	}

	ops := getTransactOpts(cNonce)

	txInfo, err := c.GetCreateSubAccountTransaction(ops)
	return convertTxInfoToResponse(txInfo, err)
}

//export SignCancelAllOrders
func SignCancelAllOrders(cTimeInForce int32, cTime int64, cNonce int64, cApiKeyIndex int32, cAccountIndex int64) (ret SignedTxResponse) {
	defer func() {
		if r := recover(); r != nil {
			ret = signedTxResponsePanic(fmt.Errorf("%v", r))
		}
	}()

	c, err := getClient(cApiKeyIndex, cAccountIndex)
	if err != nil {
		return signedTxResponseErr(err)
	}

	timeInForce := uint8(cTimeInForce)
	t := int64(cTime)

	tx := &types.CancelAllOrdersTxReq{
		TimeInForce: timeInForce,
		Time:        t,
	}
	ops := getTransactOpts(cNonce)

	txInfo, err := c.GetCancelAllOrdersTransaction(tx, ops)
	return convertTxInfoToResponse(txInfo, err)
}

//export SignModifyOrder
func SignModifyOrder(cMarketIndex int32, cIndex int64, cBaseAmount int64, cPrice int64, cTriggerPrice int64, cNonce int64, cApiKeyIndex int32, cAccountIndex int64) (ret SignedTxResponse) {
	defer func() {
		if r := recover(); r != nil {
			ret = signedTxResponsePanic(fmt.Errorf("%v", r))
		}
	}()

	c, err := getClient(cApiKeyIndex, cAccountIndex)
	if err != nil {
		return signedTxResponseErr(err)
	}

	marketIndex := int16(cMarketIndex)
	index := int64(cIndex)
	baseAmount := int64(cBaseAmount)
	price := uint32(cPrice)
	triggerPrice := uint32(cTriggerPrice)

	tx := &types.ModifyOrderTxReq{
		MarketIndex:  marketIndex,
		Index:        index,
		BaseAmount:   baseAmount,
		Price:        price,
		TriggerPrice: triggerPrice,
	}
	ops := getTransactOpts(cNonce)

	txInfo, err := c.GetModifyOrderTransaction(tx, ops)
	return convertTxInfoToResponse(txInfo, err)
}

//export SignTransfer
func SignTransfer(cToAccountIndex int64, cAssetIndex int16, cFromRouteType, cToRouteType uint8, cAmount, cUsdcFee int64, cMemo string, cNonce int64, cApiKeyIndex int32, cAccountIndex int64) (ret SignedTxResponse) {
	defer func() {
		if r := recover(); r != nil {
			ret = signedTxResponsePanic(fmt.Errorf("%v", r))
		}
	}()

	c, err := getClient(cApiKeyIndex, cAccountIndex)
	if err != nil {
		return signedTxResponseErr(err)
	}

	toAccountIndex := int64(cToAccountIndex)
	assetIndex := int16(cAssetIndex)
	fromRouteType := uint8(cFromRouteType)
	toRouteType := uint8(cToRouteType)
	amount := int64(cAmount)
	usdcFee := int64(cUsdcFee)
	memo := [32]byte{}
	memoStr := cMemo
	if len(memoStr) == 66 {
		if memoStr[0:2] == "0x" {
			memoStr = memoStr[2:66]
		} else {
			return signedTxResponseErr(fmt.Errorf("memo expected to be 32 bytes or 64 hex encoded or 66 if 0x hex encoded -- long but received %v", len(memoStr)))
		}
	}

	// assume hex encoded here
	if len(memoStr) == 64 {
		b, err := hex.DecodeString(memoStr)
		if err != nil {
			return signedTxResponseErr(fmt.Errorf("failed to decode hex string. err: %v", err))
		}

		for i := 0; i < 32; i += 1 {
			memo[i] = b[i]
		}
	} else if len(memoStr) == 32 {
		for i := 0; i < 32; i++ {
			memo[i] = byte(memoStr[i])
		}
	} else {
		return signedTxResponseErr(fmt.Errorf("memo expected to be 32 bytes or 64 hex encoded or 66 if 0x hex encoded -- long but received %v", len(memoStr)))
	}

	tx := &types.TransferTxReq{
		ToAccountIndex: toAccountIndex,
		AssetIndex:     assetIndex,
		FromRouteType:  fromRouteType,
		ToRouteType:    toRouteType,
		Amount:         amount,
		USDCFee:        usdcFee,
		Memo:           memo,
	}
	ops := getTransactOpts(cNonce)

	txInfo, err := c.GetTransferTransaction(tx, ops)
	return convertTxInfoToResponse(txInfo, err)
}

//export SignCreatePublicPool
func SignCreatePublicPool(cOperatorFee int64, cInitialTotalShares int32, cMinOperatorShareRate int64, cNonce int64, cApiKeyIndex int32, cAccountIndex int64) (ret SignedTxResponse) {
	defer func() {
		if r := recover(); r != nil {
			ret = signedTxResponsePanic(fmt.Errorf("%v", r))
		}
	}()

	c, err := getClient(cApiKeyIndex, cAccountIndex)
	if err != nil {
		return signedTxResponseErr(err)
	}

	operatorFee := int64(cOperatorFee)
	initialTotalShares := int64(cInitialTotalShares)
	minOperatorShareRate := uint16(cMinOperatorShareRate)

	tx := &types.CreatePublicPoolTxReq{
		OperatorFee:          operatorFee,
		InitialTotalShares:   initialTotalShares,
		MinOperatorShareRate: minOperatorShareRate,
	}
	ops := getTransactOpts(cNonce)

	txInfo, err := c.GetCreatePublicPoolTransaction(tx, ops)
	return convertTxInfoToResponse(txInfo, err)
}

//export SignUpdatePublicPool
func SignUpdatePublicPool(cPublicPoolIndex int64, cStatus int32, cOperatorFee int64, cMinOperatorShareRate int32, cNonce int64, cApiKeyIndex int32, cAccountIndex int64) (ret SignedTxResponse) {
	defer func() {
		if r := recover(); r != nil {
			ret = signedTxResponsePanic(fmt.Errorf("%v", r))
		}
	}()

	c, err := getClient(cApiKeyIndex, cAccountIndex)
	if err != nil {
		return signedTxResponseErr(err)
	}

	publicPoolIndex := int64(cPublicPoolIndex)
	status := uint8(cStatus)
	operatorFee := int64(cOperatorFee)
	minOperatorShareRate := uint16(cMinOperatorShareRate)

	tx := &types.UpdatePublicPoolTxReq{
		PublicPoolIndex:      publicPoolIndex,
		Status:               status,
		OperatorFee:          operatorFee,
		MinOperatorShareRate: minOperatorShareRate,
	}
	ops := getTransactOpts(cNonce)

	txInfo, err := c.GetUpdatePublicPoolTransaction(tx, ops)
	return convertTxInfoToResponse(txInfo, err)
}

//export SignMintShares
func SignMintShares(cPublicPoolIndex int64, cShareAmount int64, cNonce int64, cApiKeyIndex int32, cAccountIndex int64) (ret SignedTxResponse) {
	defer func() {
		if r := recover(); r != nil {
			ret = signedTxResponsePanic(fmt.Errorf("%v", r))
		}
	}()

	c, err := getClient(cApiKeyIndex, cAccountIndex)
	if err != nil {
		return signedTxResponseErr(err)
	}

	publicPoolIndex := int64(cPublicPoolIndex)
	shareAmount := int64(cShareAmount)

	tx := &types.MintSharesTxReq{
		PublicPoolIndex: publicPoolIndex,
		ShareAmount:     shareAmount,
	}
	ops := getTransactOpts(cNonce)

	txInfo, err := c.GetMintSharesTransaction(tx, ops)
	return convertTxInfoToResponse(txInfo, err)
}

//export SignBurnShares
func SignBurnShares(cPublicPoolIndex int64, cShareAmount int64, cNonce int64, cApiKeyIndex int32, cAccountIndex int64) (ret SignedTxResponse) {
	defer func() {
		if r := recover(); r != nil {
			ret = signedTxResponsePanic(fmt.Errorf("%v", r))
		}
	}()

	c, err := getClient(cApiKeyIndex, cAccountIndex)
	if err != nil {
		return signedTxResponseErr(err)
	}

	publicPoolIndex := int64(cPublicPoolIndex)
	shareAmount := int64(cShareAmount)

	tx := &types.BurnSharesTxReq{
		PublicPoolIndex: publicPoolIndex,
		ShareAmount:     shareAmount,
	}
	ops := getTransactOpts(cNonce)

	txInfo, err := c.GetBurnSharesTransaction(tx, ops)
	return convertTxInfoToResponse(txInfo, err)
}

//export SignUpdateLeverage
func SignUpdateLeverage(cMarketIndex int32, cInitialMarginFraction int32, cMarginMode int32, cNonce int64, cApiKeyIndex int32, cAccountIndex int64) (ret SignedTxResponse) {
	defer func() {
		if r := recover(); r != nil {
			ret = signedTxResponsePanic(fmt.Errorf("%v", r))
		}
	}()

	c, err := getClient(cApiKeyIndex, cAccountIndex)
	if err != nil {
		return signedTxResponseErr(err)
	}

	marketIndex := int16(cMarketIndex)
	initialMarginFraction := uint16(cInitialMarginFraction)
	marginMode := uint8(cMarginMode)

	tx := &types.UpdateLeverageTxReq{
		MarketIndex:           marketIndex,
		InitialMarginFraction: initialMarginFraction,
		MarginMode:            marginMode,
	}
	ops := getTransactOpts(cNonce)

	txInfo, err := c.GetUpdateLeverageTransaction(tx, ops)
	return convertTxInfoToResponse(txInfo, err)
}

//export CreateAuthToken
func CreateAuthToken(cDeadline int64, cApiKeyIndex int32, cAccountIndex int64) (retStr string, retErr error) {
	defer func() {
		if r := recover(); r != nil {
			retErr = fmt.Errorf("panic: %v", r)
		}
	}()

	c, err := getClient(cApiKeyIndex, cAccountIndex)
	if err != nil {
		return "", err
	}

	deadline := int64(cDeadline)
	if deadline == 0 {
		deadline = time.Now().Add(time.Hour * 7).Unix()
	}

	authToken, err := c.GetAuthToken(time.Unix(deadline, 0))
	if err != nil {
		return "", err
	}

	return authToken, nil
}

//export SignUpdateMargin
func SignUpdateMargin(cMarketIndex int32, cUSDCAmount int64, cDirection int32, cNonce int64, cApiKeyIndex int32, cAccountIndex int64) (ret SignedTxResponse) {
	defer func() {
		if r := recover(); r != nil {
			ret = signedTxResponsePanic(fmt.Errorf("%v", r))
		}
	}()

	c, err := getClient(cApiKeyIndex, cAccountIndex)
	if err != nil {
		return signedTxResponseErr(err)
	}

	marketIndex := int16(cMarketIndex)
	usdcAmount := int64(cUSDCAmount)
	direction := uint8(cDirection)

	tx := &types.UpdateMarginTxReq{
		MarketIndex: marketIndex,
		USDCAmount:  usdcAmount,
		Direction:   direction,
	}
	ops := getTransactOpts(cNonce)

	txInfo, err := c.GetUpdateMarginTransaction(tx, ops)
	return convertTxInfoToResponse(txInfo, err)
}
