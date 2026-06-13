package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	shell "github.com/ipfs/go-ipfs-api"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	ethclient "github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/common"
)

// Request payload from Rust service
type MintRequest struct {
	ToAddress     string `json:"to_address"`
	IpfsCid       string `json:"ipfs_cid"`
	CheckpointHex string `json:"checkpoint"` // hex string of checkpoint hash (32 bytes)
	Nonce         *uint64 `json:"nonce,omitempty"`
}

// environment vars:
// GETH_RPC (http://geth:8545), PRIVATE_KEY_PATH (path inside container), CONTRACT_ADDRESS
func MintHandler(c *gin.Context) {
	var req MintRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	geth := os.Getenv("GETH_RPC")
	if geth == "" { geth = "http://geth:8545" }
	contractAddr := os.Getenv("CONTRACT_ADDRESS")
	if contractAddr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error":"CONTRACT_ADDRESS not set in env"})
		return
	}

	// 1) Validate/resolve IPFS CID — optional: ping IPFS API
	ipfs := shell.NewShell("http://ipfs:5001")
	_, err := ipfs.ObjectStat(req.IpfsCid)
	if err != nil {
		// not found or error
		log.Printf("ipfs stat error: %v", err)
		// proceed anyway
	}

	// 2) connect to geth
	client, err := ethclient.Dial(geth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed dial geth: " + err.Error()})
		return
	}

	// 3) load private key
	privPath := os.Getenv("PRIVATE_KEY_PATH")
	privKeyBytes, err := ioutil.ReadFile(privPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error":"failed read private key:" + err.Error()})
		return
	}
	privKeyHex := string(privKeyBytes)
	privKeyHex = trimNewline(privKeyHex)
	privKey, err := crypto.HexToECDSA(privKeyHex)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error":"invalid private key: " + err.Error()})
		return
	}
	publicKey := privKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error":"invalid public key"})
		return
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// 4) prepare auth
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "fail network id: " + err.Error()})
		return
	}
	auth, err := bind.NewKeyedTransactorWithChainID(privKey, chainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error":"transactor error: " + err.Error()})
		return
	}

	// optional: set gas params / nonce
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// load contract ABI & address
	contractAddress := common.HexToAddress(contractAddr)
	// Use generated Go bindings ideally; here we use generic ABI binding call
	// For simplicity, we will call via low-level Transact with ABI encoded data.
	abiBytes, err := ioutil.ReadFile("contract/StoryNFT.abi.json")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error":"abi read error: " + err.Error()})
		return
	}
	// build ABI method call
	parsedABI, err := abi.JSON(strings.NewReader(string(abiBytes)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error":"abi parse error: "+err.Error()})
		return
	}

	// pack inputs: mintStory(address to, string cid, bytes32 checkpoint)
	toAddr := common.HexToAddress(req.ToAddress)
	checkpointBytes, err := hex.DecodeString(trimHexPrefix(req.CheckpointHex))
	if err != nil || len(checkpointBytes) != 32 {
		c.JSON(http.StatusBadRequest, gin.H{"error":"checkpoint must be 32-byte hex"})
		return
	}
	input, err := parsedABI.Pack("mintStory", toAddr, req.IpfsCid, [32]byte{})
	if err != nil {
		// build proper 32-byte array
		var cp [32]byte
		copy(cp[:], checkpointBytes)
		input, err = parsedABI.Pack("mintStory", toAddr, req.IpfsCid, cp)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error":"abi pack fail: "+err.Error()})
			return
		}
	}

	nonce := uint64(0)
	if req.Nonce != nil {
		nonce = *req.Nonce
	} else {
		nonce64, err := client.PendingNonceAt(ctx, fromAddress)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error":"nonce fail: "+err.Error()})
			return
		}
		nonce = nonce64
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error":"gas price fail:"+err.Error()})
		return
	}
	tx := types.NewTransaction(nonce, contractAddress, big.NewInt(0), uint64(300000), gasPrice, input)
	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), privKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error":"sign tx fail:"+err.Error()})
		return
	}

	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error":"send tx fail:"+err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"tx_hash": signedTx.Hash().Hex(),
		"from": fromAddress.Hex(),
	})
}

