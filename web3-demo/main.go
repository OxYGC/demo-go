package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// UTXO 结构体用于解析 API 返回
type UTXO struct {
	Txid  string `json:"txid"`
	Vout  uint32 `json:"vout"`
	Value int64  `json:"value"`
}

func main() {
	// 使用测试网配置，生产环境请用 MainNetParams
	netParams := &chaincfg.TestNet3Params

	fmt.Println("--- 1. 生成公私钥对 ---")
	privKey, addr, err := generateWallet(netParams)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("私钥 (Hex): %x\n", privKey.Serialize())
	fmt.Printf("地址 (SegWit): %s\n", addr.EncodeAddress())

	fmt.Println("\n--- 2. 签名示例 ---")
	message := "Hello BTC Web3"
	sig, err := signMessage(privKey, message)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("消息签名: %s\n", sig)

	fmt.Println("\n--- 3. 获取链上报文 (模拟获取 UTXO) ---")
	// 注意：这里需要一个真实的测试网地址来获取数据，演示时我们打印流程
	utxos, err := fetchUTXOs(addr.EncodeAddress())
	if err != nil {
		fmt.Printf("获取 UTXO 失败 (可能地址太新): %v\n", err)
	} else {
		fmt.Printf("找到 %d 个可用 UTXO\n", len(utxos))
	}

	fmt.Println("\n--- 4. 构造提现交易并模拟上链 ---")
	// 这里演示构造一个简单的 P2WPKH 交易
	rawTx, err := constructWithdrawal(privKey, addr, "tb1qw508d6qejxtdg4y5r3zarvary0c5xw7kxpjzsx", 1000, netParams)
	if err != nil {
		fmt.Printf("构造交易失败 (无足够余额): %v\n", err)
	} else {
		fmt.Printf("构造的原始交易 (Hex): %s\n", rawTx)
	}
}

// generateWallet 生成一个新的比特币地址 (SegWit P2WPKH)
func generateWallet(params *chaincfg.Params) (*btcec.PrivateKey, *btcutil.AddressWitnessPubKeyHash, error) {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}

	pubKeyHash := btcutil.Hash160(privKey.PubKey().SerializeCompressed())
	addr, err := btcutil.NewAddressWitnessPubKeyHash(pubKeyHash, params)
	if err != nil {
		return nil, nil, err
	}

	return privKey, addr, nil
}

// signMessage 对普通消息进行签名
func signMessage(privKey *btcec.PrivateKey, message string) (string, error) {
	hash := chainhash.DoubleHashB([]byte(message))
	sig := btcec.Signature{}                                                                // 实际中常用 Schnorr 或 ECDSA
	signature := btcec.NewSignature(privKey.Key.ToModNScalar(), privKey.Key.ToModNScalar()) // 简化示例
	_ = signature                                                                           // 占位

	// 在生产中，我们会使用 privKey.Sign(hash)
	signed, err := privKey.Sign(hash)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(signed.Serialize()), nil
}

// fetchUTXOs 从 mempool.space 获取地址的 UTXO
func fetchUTXOs(address string) ([]UTXO, error) {
	url := fmt.Sprintf("https://mempool.space/testnet/api/address/%s/utxo", address)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var utxos []UTXO
	err = json.Unmarshal(body, &utxos)
	return utxos, err
}

// constructWithdrawal 构造并签署一个提现交易
func constructWithdrawal(privKey *btcec.PrivateKey, fromAddr *btcutil.AddressWitnessPubKeyHash, toAddressStr string, amount int64, params *chaincfg.Params) (string, error) {
	// 1. 准备目标地址
	toAddr, err := btcutil.DecodeAddress(toAddressStr, params)
	if err != nil {
		return "", err
	}
	toPkScript, _ := txscript.PayToAddrScript(toAddr)

	// 2. 创建一个新的交易对象
	msgTx := wire.NewMsgTx(wire.TxVersion)

	// 3. 添加输出 (提现金额)
	msgTx.AddTxOut(wire.NewTxOut(amount, toPkScript))

	// 4. 这里需要添加 Input (UTXO)。为了演示，我们假设有一个虚拟的 UTXO
	// 在实际逻辑中，你需要遍历 fetchUTXOs 的结果并添加
	hash, _ := chainhash.NewHashFromStr("0000000000000000000000000000000000000000000000000000000000000000")
	outPoint := wire.NewOutPoint(hash, 0)
	txIn := wire.NewTxIn(outPoint, nil, nil)
	msgTx.AddTxIn(txIn)

	// 5. 签名逻辑 (SegWit)
	// 注意：SegWit 签名需要知道 Input 的金额，这里仅展示框架
	// 实际签名会使用 txscript.WitnessSignature

	var buf bytes.Buffer
	if err := msgTx.Serialize(&buf); err != nil {
		return "", err
	}

	return hex.EncodeToString(buf.Bytes()), nil
}
