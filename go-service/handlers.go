package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	shell "github.com/ipfs/go-ipfs-api"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	ethclient "github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/core/types"

	"go-service/contract" // generated via abigen -> package contract
)

type MintRequest struct {
	ToAddress     string `json:"to_address"`
	IpfsCid       string `json:"ipfs_cid"`
	CheckpointHex string `json:"checkpoint"` // 64 hex chars
	Nonce         *uint64 `json:"nonce,omitempty"`
}

func trimHexPrefix(s string) string {
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		return s[2:]
	}
	return s
}

func trimNewline(s string) string {
	return strings.TrimSpace(s)
}

func MintHandler(c *gin.Context) {
	var req MintRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	geth := os.Getenv("GETH_RPC")
	if geth == "" { geth = "http://geth:8545" }
	contractAddr := os.Getenv("CONTRACT_ADDRESS")
	if contractAddr == "" {
		c.JSON(500, gin.H{"error":"CONTRACT_ADDRESS not set in env"})
		return
	}

	// optional: ping IPFS
	ipfs := shell.NewShell("http://ipfs:5001")
	_, err := ipfs.ObjectStat(req.IpfsCid)
	if err != nil {
		log.Printf("ipfs stat error (continue): %v", err)
	}

	client, err := ethclient.Dial(geth)
	if err != nil {
		c.JSON(500, gin.H{"error":"failed dial geth: " + err.Error()})
		return
	}

	// load private key
	privPath := os.Getenv("PRIVATE_KEY_PATH")
	privKeyBytes, err := ioutil.ReadFile(privPath)
	if err != nil {
		c.JSON(500, gin.H{"error":"failed read private key:" + err.Error()})
		return
	}
	privKeyHex := trimNewline(string(privKeyBytes))
	privKey, err := crypto.HexToECDSA(privKeyHex)
	if err != nil {
		c.JSON(500, gin.H{"error":"invalid private key: " + err.Error()})
		return
	}
	publicKey := privKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		c.JSON(500, gin.H{"error":"invalid public key"})
		return
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	chainID, err := client.NetworkID(ctx)
	if err != nil {
		c.JSON(500, gin.H{"error":"fail network id: " + err.Error()})
		return
	}

	// prepare transactor
	auth, err := bind.NewKeyedTransactorWithChainID(privKey, chainID)
	if err != nil {
		c.JSON(500, gin.H{"error":"transactor error: " + err.Error()})
		return
	}

	// set nonce/gasprice if needed
	if req.Nonce != nil {
		auth.Nonce = big.NewInt(int64(*req.Nonce))
	} else {
		nonce, err := client.PendingNonceAt(ctx, fromAddress)
		if err != nil {
			c.JSON(500, gin.H{"error":"nonce fetch fail: " + err.Error()})
			return
		}
		auth.Nonce = big.NewInt(int64(nonce))
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err == nil {
		auth.GasPrice = gasPrice
	}
	auth.GasLimit = uint64(300000)

	// load contract binding
	contractAddress := common.HexToAddress(contractAddr)
	story, err := contract.NewStoryNFT(contractAddress, client)
	if err != nil {
		c.JSON(500, gin.H{"error":"failed contract binding: " + err.Error()})
		return
	}

	// prepare checkpoint bytes32
	cpHex := trimHexPrefix(req.CheckpointHex)
	cpBytes, err := hex.DecodeString(cpHex)
	if err != nil || len(cpBytes) != 32 {
		c.JSON(400, gin.H{"error":"checkpoint must be 32-byte hex (64 hex chars)"})
		return
	}
	var cpArr [32]byte
	copy(cpArr[:], cpBytes)

	toAddr := common.HexToAddress(req.ToAddress)
	tx, err := story.MintStory(auth, toAddr, req.IpfsCid, cpArr)
	if err != nil {
		c.JSON(500, gin.H{"error":"mint tx failed: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"tx_hash": tx.Hash().Hex(),
		"from": fromAddress.Hex(),
	})
}
