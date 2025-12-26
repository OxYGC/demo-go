package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config 结构体，用于解析 YAML 配置文件
type Config struct {
	WalletNode struct {
		BTC NodeConfig `yaml:"btc"`
		ETH NodeConfig `yaml:"eth"`
	} `yaml:"wallet_node"`
}

type NodeConfig struct {
	RPCURL       string `yaml:"rpc_url"`
	RPCUser      string `yaml:"rpc_user"`
	RPCPass      string `yaml:"rpc_pass"`
	DataAPIURL   string `yaml:"data_api_url"`
	DataAPIKey   string `yaml:"data_api_key"`
	DataAPIToken string `yaml:"data_api_token"`
	Timeout      int    `yaml:"time_out"`
}

var globalConfig NodeConfig

// 加载配置：先从 .env 获取，如果没有则从 config.yml 获取
func loadConfig() {
	// 加载 .env
	_ = godotenv.Load()

	// 1. 尝试从 .env 获取
	nodeURL := os.Getenv("BTC_NODE_API_URL")
	if nodeURL != "" {
		globalConfig.DataAPIURL = nodeURL
		globalConfig.RPCURL = nodeURL // 简单起见，这里共用
		return
	}

	// 2. 如果 .env 没有，从 config.yml 获取
	yamlFile, err := os.ReadFile("config.yml")
	if err == nil {
		var cfg Config
		err = yaml.Unmarshal(yamlFile, &cfg)
		if err == nil && cfg.WalletNode.BTC.DataAPIURL != "" {
			globalConfig = cfg.WalletNode.BTC
			return
		}
	}

	// 3. 默认值 fallback
	if globalConfig.DataAPIURL == "" {
		globalConfig.DataAPIURL = "https://blockstream.info/testnet/api"
		globalConfig.RPCURL = "https://blockstream.info/testnet/api"
	}
}

// UTXO 结构体，用于解析链上报文
type UTXO struct {
	TxID   string `json:"txid"`
	Vout   uint32 `json:"vout"`
	Status struct {
		Confirmed bool `json:"confirmed"`
	} `json:"status"`
	Value int64 `json:"value"`
}

// 1. 生成公私钥和地址
func GenerateKey(net *chaincfg.Params) (*btcec.PrivateKey, *btcutil.AddressPubKey, error) {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}

	pubKey := privKey.PubKey()
	addrPubKey, err := btcutil.NewAddressPubKey(pubKey.SerializeCompressed(), net)
	if err != nil {
		return nil, nil, err
	}

	return privKey, addrPubKey, nil
}

// 2. 签名方法
func SignMessage(privKey *btcec.PrivateKey, message string) (string, error) {
	hash := chainhash.DoubleHashB([]byte(message))
	signature := ecdsa.Sign(privKey, hash)
	return hex.EncodeToString(signature.Serialize()), nil
}

// 3. 获取链上报文 (UTXO) - 使用 Blockstream API 示例
func GetUTXOs(address string) ([]UTXO, error) {
	nodeURL := globalConfig.DataAPIURL
	// 这里使用 blockstream.info 的测试网 API
	url := fmt.Sprintf("%s/address/%s/utxo", nodeURL, address)

	client := &http.Client{}
	if globalConfig.Timeout > 0 {
		client.Timeout = time.Duration(globalConfig.Timeout) * time.Second
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// 添加 API Key/Token (如果有)
	if globalConfig.DataAPIKey != "" {
		req.Header.Add("X-API-Key", globalConfig.DataAPIKey)
	}
	if globalConfig.DataAPIToken != "" {
		req.Header.Add("Authorization", "Bearer "+globalConfig.DataAPIToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var utxos []UTXO
	err = json.Unmarshal(body, &utxos)
	if err != nil {
		return nil, err
	}

	return utxos, nil
}

// 4. 提现构造并签名
func CreateTransaction(privKey *btcec.PrivateKey, fromAddr btcutil.Address, toAddrStr string, amount int64, utxos []UTXO, net *chaincfg.Params) (*wire.MsgTx, error) {
	// 构造目的地址
	toAddr, err := btcutil.DecodeAddress(toAddrStr, net)
	if err != nil {
		return nil, err
	}
	toPkScript, _ := txscript.PayToAddrScript(toAddr)

	// 创建新的交易
	tx := wire.NewMsgTx(wire.TxVersion)

	var totalInput int64 = 0
	for _, utxo := range utxos {
		hash, _ := chainhash.NewHashFromStr(utxo.TxID)
		outPoint := wire.NewOutPoint(hash, utxo.Vout)
		txIn := wire.NewTxIn(outPoint, nil, nil)
		tx.AddTxIn(txIn)
		totalInput += utxo.Value
		if totalInput >= amount+1000 { // 1000 satoshi 作为简单手续费
			break
		}
	}

	if totalInput < amount+1000 {
		return nil, fmt.Errorf("insufficient funds")
	}

	// 添加输出
	txOut := wire.NewTxOut(amount, toPkScript)
	tx.AddTxOut(txOut)

	// 找零
	change := totalInput - amount - 1000
	if change > 0 {
		changePkScript, _ := txscript.PayToAddrScript(fromAddr)
		tx.AddTxOut(wire.NewTxOut(change, changePkScript))
	}

	// 签名
	for i := range tx.TxIn {
		// 这里假设是 P2PKH 脚本，实际需根据 UTXO 类型动态获取脚本
		fromPkScript, _ := txscript.PayToAddrScript(fromAddr)
		sigScript, err := txscript.SignatureScript(tx, i, fromPkScript, txscript.SigHashAll, privKey, true)
		if err != nil {
			return nil, err
		}
		tx.TxIn[i].SignatureScript = sigScript
	}

	return tx, nil
}

// 5. 广播交易
func BroadcastTransaction(tx *wire.MsgTx) (string, error) {
	var buf bytes.Buffer
	if err := tx.Serialize(&buf); err != nil {
		return "", err
	}
	txHex := hex.EncodeToString(buf.Bytes())

	nodeURL := globalConfig.DataAPIURL
	// 使用 blockstream.info 的 API 广播
	url := fmt.Sprintf("%s/tx", nodeURL)

	client := &http.Client{}
	if globalConfig.Timeout > 0 {
		client.Timeout = time.Duration(globalConfig.Timeout) * time.Second
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader([]byte(txHex)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "text/plain")

	// 添加 API Key/Token (如果有)
	if globalConfig.DataAPIKey != "" {
		req.Header.Add("X-API-Key", globalConfig.DataAPIKey)
	}
	if globalConfig.DataAPIToken != "" {
		req.Header.Add("Authorization", "Bearer "+globalConfig.DataAPIToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
}

func main() {
	// 加载配置
	loadConfig()

	net := &chaincfg.TestNet3Params // 使用测试网

	// 从环境变量获取私钥和地址
	privKeyHex := os.Getenv("BTC_PRIVATE_KEY")
	addressStr := os.Getenv("BTC_ADDRESS")

	if privKeyHex == "" || addressStr == "" {
		log.Fatal("BTC_PRIVATE_KEY or BTC_ADDRESS not set in .env")
	}

	// 解析私钥
	privKeyBytes, err := hex.DecodeString(privKeyHex)
	if err != nil {
		log.Fatalf("Invalid private key hex: %v", err)
	}
	privKey, _ := btcec.PrivKeyFromBytes(privKeyBytes)

	// 解析地址
	address, err := btcutil.DecodeAddress(addressStr, net)
	if err != nil {
		log.Fatalf("Invalid address: %v", err)
	}

	fmt.Printf("Loaded Address: %s\n", address.String())

	// B. 签名示例
	sig, _ := SignMessage(privKey, "Hello BTC")
	fmt.Printf("Signature for 'Hello BTC': %s\n", sig)

	// C. 获取 UTXO
	fmt.Println("Fetching UTXOs...")
	utxos, err := GetUTXOs(address.String())
	if err != nil {
		fmt.Printf("Error fetching UTXOs: %v\n", err)
	} else {
		fmt.Printf("Found %d UTXOs\n", len(utxos))
	}

	// D. 构造并签名交易
	if len(utxos) > 0 {
		destAddr := "tb1qw508d6qejxtdg4y5r3zarvary0c5xw7kxpjzsx" // 测试网地址示例
		tx, err := CreateTransaction(privKey, address, destAddr, 1000, utxos, net)
		if err != nil {
			fmt.Printf("Error creating tx: %v\n", err)
		} else {
			fmt.Println("Transaction created and signed.")
			// E. 广播
			// txHash, err := BroadcastTransaction(tx)
			// fmt.Printf("Broadcast result: %s, error: %v\n", txHash, err)
			_ = tx
		}
	} else {
		fmt.Println("No UTXOs found, skipping transaction creation.")
	}
}
