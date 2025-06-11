package main

import (
  "fmt"
  "encoding/json"
  "log"
  "github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// SmartContract provides functions for managing an Asset
type SmartContract struct {
  contractapi.Contract
}

// Asset describes basic details of what makes up a simple asset
// Insert struct field in alphabetic order => to achieve determinism across languages
// golang keeps the order when marshal to json but doesn't order automatically

type Asset struct {
  Author		     string `json:"Author"`
  Date           string `json:"Date"`
  ID             int    `json:"ID"`
  Message        string `json:"Message"`
}

// InitLedger adds a base set of assets to the ledger
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
  assets := []Asset{
    {ID: 0, Author: "user_0", Date: "Unknown", Message: "Unknown"},
  }

  for _, asset := range assets {
    assetJSON, err := json.Marshal(asset)
    if err != nil {
        return err
    }

    err = ctx.GetStub().PutState(fmt.Sprint(asset.ID), assetJSON)
    if err != nil {
        return fmt.Errorf("failed to put to world state. %v", err)
    }
  }

  return nil
}

// AssetExists returns true when asset with given ID exists in world state
func (s *SmartContract) AssetExists(ctx contractapi.TransactionContextInterface, id int) (bool, error) {
  assetJSON, err := ctx.GetStub().GetState(fmt.Sprint(id))
  if err != nil {
    return false, fmt.Errorf("failed to read from world state: %v", err)
  }

  return assetJSON != nil, nil
}

// CreateAsset issues a new asset to the world state with given details.
func (s *SmartContract) CreateAsset(ctx contractapi.TransactionContextInterface, id int, author string, date string, message string) error {
  exists, err := s.AssetExists(ctx, id)
  if err != nil {
    return err
  }
  if exists {
    return fmt.Errorf("the asset %d already exists", id)
  }

  asset := Asset{
    ID:             id,
    Author:         author,
    Date:           date,
    Message:        message,
  }
  assetJSON, err := json.Marshal(asset)
  if err != nil {
    return err
  }

  return ctx.GetStub().PutState(fmt.Sprint(id), assetJSON)
}

// ReadAsset returns the asset stored in the world state with given id.
func (s *SmartContract) ReadAsset(ctx contractapi.TransactionContextInterface, id int) (*Asset, error) {
  assetJSON, err := ctx.GetStub().GetState(fmt.Sprint(id))
  if err != nil {
    return nil, fmt.Errorf("failed to read from world state: %v", err)
  }
  if assetJSON == nil {
    return nil, fmt.Errorf("the asset %d does not exist", id)
  }

  var asset Asset
  err = json.Unmarshal(assetJSON, &asset)
  if err != nil {
    return nil, err
  }

  return &asset, nil
}

// UpdateAsset updates an existing asset in the world state with provided parameters.
func (s *SmartContract) UpdateAsset(ctx contractapi.TransactionContextInterface, id int, author string, date string, message string) error {
  exists, err := s.AssetExists(ctx, id)
  if err != nil {
    return err
  }
  if !exists {
    return fmt.Errorf("the asset %d does not exist", id)
  }

  // overwriting original asset with new asset
  asset := Asset{
    ID:             id,
    Author:         author,
    Date:           date,
    Message:        message,
  }
  assetJSON, err := json.Marshal(asset)
  if err != nil {
    return err
  }

  return ctx.GetStub().PutState(fmt.Sprint(id), assetJSON)
}

func main() {
  assetChaincode, err := contractapi.NewChaincode(&SmartContract{})
  if err != nil {
    log.Panicf("Error creating chaincode: %v", err)
  }

  if err := assetChaincode.Start(); err != nil {
    log.Panicf("Error starting chaincode: %v", err)
  }
}