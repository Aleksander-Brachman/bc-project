/* 
Skrypt stworozny na bazie kodu dostarczonego przez twórców HF:
https://github.com/hyperledger/fabric-samples/blob/main/asset-transfer-basic/application-gateway-go/assetTransfer.go

*/

package main

import ( // Importowanie modułów GO
	"bytes"
	//"context"
	"crypto/x509"
	"encoding/json"
	//"errors"
	"fmt"
	"log"
	"os"
	"path"
	"time"
	"database/sql"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/hash"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	//"github.com/hyperledger/fabric-protos-go-apiv2/gateway"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	//"google.golang.org/grpc/status"
	"github.com/go-sql-driver/mysql"
)

const ( // Stałe zmienne potrzebne do komunikacji z siecią BC
	mspID        = "Org1MSP"
	cryptoPath   = "/home/testdebian/fabric/fabric-samples/test-network/organizations/peerOrganizations/org1.example.com"
	certPath     = cryptoPath + "/users/User1@org1.example.com/msp/signcerts"
	keyPath      = cryptoPath + "/users/User1@org1.example.com/msp/keystore"
	tlsCertPath  = cryptoPath + "/peers/peer0.org1.example.com/tls/ca.crt"
	peerEndpoint = "dns:///localhost:7051"
	gatewayPeer  = "peer0.org1.example.com"
)

func main() { // Inicjalizacja połączenia z HF i wykonywanie skryptu
	// The gRPC client connection should be shared by all Gateway connections to this endpoint
	clientConnection := newGrpcConnection()
	defer clientConnection.Close()

	id := newIdentity()
	sign := newSign()

	// Create a Gateway connection for a specific client identity
	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithHash(hash.SHA256),
		client.WithClientConnection(clientConnection),
		// Default timeouts for different gRPC calls
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer gw.Close()

	// Nazwy smartcontractu i kanału w HF
	chaincodeName := "message_sc"
	channelName := "mychannel"


	network := gw.GetNetwork(channelName)
	contract := network.GetContract(chaincodeName)

	// Połączenie z MariaDB
	db, err := connToMariaDB()
	if err != nil {
		fmt.Errorf("Error connecting to database: %v", err)
	}
	defer db.Close()

	// Ustawienie odliczania (ticku) co 5 sekund
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Co tick (co 5 sekund) pobieraj rekordy z SQL
	for range ticker.C {
		records, err := fetchRecords(db)
		if err != nil {
			fmt.Errorf("Error fetching records: %v", err)
			continue
		}
		if len(records) == 0 { // Jeżeli tablica rekordów SQL jest pusta to skrypt czeka do następnego ticku 
			fmt.Println("No records found")
            continue
		}
		for _, record := range records { // Przetwarzanie tablicy rekordów SQL rekord po rekordzie
			fmt.Printf("Record ID: %d, Date: %s", record.ID, record.Date)
			exists, err := assetExists(contract, record.ID) // Weryfikacja czy dana wiadomość (jej ID) jest już w ledgerze
			if err != nil {
				fmt.Errorf("assetExists error: %v", err)
				continue
			}

			if !exists { // Jeżeli nie, to następuje utworzenie nowego assetu wiadomości z oryginalnym autorem
				err := createAsset(contract, record)
				if err != nil {
					fmt.Errorf("createAsset error: %v", err)
				}
			} else { // Jeżeli tak, to następuje weryfikacja czy autor aktualizacji wiadomości jest zgodny z zapisem w ledgerze
				asset, err := readAsset(contract, record.ID)
				if err != nil {
					fmt.Errorf("readAsset error: %v", err)
					continue
				}
				if asset.Author == record.Author { // Jeżeli autorzy są zgodni to następuje aktualizacja assetu wiadomości o danym ID w ledgerze
					err := updateAsset(contract, record)
					if err != nil {
						fmt.Errorf("updateAsset error: %v", err)
					}
				} else { // Jeżeli autorzy są różni to następuje przywrócenie rekordu SQL do stanu sprzed aktualizacji wykorzystując dane z ledgera
					err := updateDatabase(db, *asset)
					if err != nil {
						fmt.Errorf("updateSQL error: %v", err)
					}
				}
			}
		}
	}
}



// Trzy funkcje konieczne do ustanowienia połączenia z HF
// newGrpcConnection creates a gRPC connection to the Gateway server.
func newGrpcConnection() *grpc.ClientConn {
	certificatePEM, err := os.ReadFile(tlsCertPath)
	if err != nil {
		panic(fmt.Errorf("failed to read TLS certifcate file: %w", err))
	}

	certificate, err := identity.CertificateFromPEM(certificatePEM)
	if err != nil {
		panic(err)
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(certificate)
	transportCredentials := credentials.NewClientTLSFromCert(certPool, gatewayPeer)

	connection, err := grpc.NewClient(peerEndpoint, grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		panic(fmt.Errorf("failed to create gRPC connection: %w", err))
	}

	return connection
}

// newIdentity creates a client identity for this Gateway connection using an X.509 certificate.
func newIdentity() *identity.X509Identity {
	certificatePEM, err := readFirstFile(certPath)
	if err != nil {
		panic(fmt.Errorf("failed to read certificate file: %w", err))
	}

	certificate, err := identity.CertificateFromPEM(certificatePEM)
	if err != nil {
		panic(err)
	}

	id, err := identity.NewX509Identity(mspID, certificate)
	if err != nil {
		panic(err)
	}

	return id
}

// newSign creates a function that generates a digital signature from a message digest using a private key.
func newSign() identity.Sign {
	privateKeyPEM, err := readFirstFile(keyPath)
	if err != nil {
		panic(fmt.Errorf("failed to read private key file: %w", err))
	}

	privateKey, err := identity.PrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		panic(err)
	}

	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		panic(err)
	}

	return sign
}




// Własne funkcje do połączenia z bazą danych i wykonywania funkcji z SC
type Record struct {
    ID         int
    Author     string
    Date       string
    Message    string
}

type Asset struct {
	ID      int    `json:"ID"`
	Author  string `json:"Author"`
	Date    string `json:"Date"`
	Message string `json:"Message"`
}

func connToMariaDB() (*sql.DB, error) { // Funkcja ustanawiająca połączenie z SQL
    cfg := mysql.NewConfig()
    cfg.User = "root"
    cfg.Passwd = "test"
    cfg.Net = "tcp"
    cfg.Addr = "localhost:3306"
    cfg.DBName = "project_db"

    db, err := sql.Open("mysql", cfg.FormatDSN())
    if err != nil {
        return nil, fmt.Errorf("*** Failed to connect to database: %w", err)
    }

    pingErr := db.Ping()
    if pingErr != nil {
        return nil, fmt.Errorf("*** Failed to ping database: %w", pingErr)
    }

    fmt.Println("*** Connected to database!")
	return db, nil
}

func fetchRecords(db *sql.DB) ([]Record, error) { // Funkcja pobierająca najnowsze rekordy z SQL i zwracająca tablicę z rekordami
    rows, err := db.Query("SELECT id, author, date, msg FROM announcement WHERE date >= SUBTIME(current_timestamp,'0 0:0:5.000000');")
    if err != nil {
        return nil, fmt.Errorf("*** Failed to fetch records: %w", err)
    }
    defer rows.Close()

    var records []Record
    for rows.Next() {
        var record Record
        err := rows.Scan(&record.ID, &record.Author, &record.Date, &record.Message)
		if err != nil {
            return nil, fmt.Errorf("*** Failed to scan record: %w", err)
        }
        records = append(records, record)
    }
    return records, nil
}

func assetExists(contract *client.Contract, id int) (bool, error) { // Funkcja sprawdzająca czy asset o danym ID już istnieje w ledgerze
	asset, err := contract.SubmitTransaction("AssetExists", fmt.Sprint(id))
	if err != nil {
		return false, fmt.Errorf("*** Failed to submit transaction assetExists: %w", err)
	}

	var exists bool
	err = json.Unmarshal(asset, &exists)
	if err != nil {
		return false, fmt.Errorf("*** Failed to unmarshal result: %w", err)
	}

	return exists, nil
}

func createAsset(contract *client.Contract, record Record) error { // Funkcja tworząca asset w ledgerze na bazie wartości z rekordu SQL
	_, err := contract.SubmitTransaction("CreateAsset", fmt.Sprint(record.ID), record.Author, record.Date, record.Message)
	if err != nil {
		return fmt.Errorf("*** Failed to submit transaction createAsset: %w", err)
	}

	fmt.Println("*** Transaction createAsset committed successfully")
	return nil
}

func readAsset(contract *client.Contract, id int) (*Asset, error) { // Funkcja, która zwraca wartości assetu o danym ID
	r_asset, err := contract.EvaluateTransaction("ReadAsset", fmt.Sprint(id))
	if err != nil {
		return nil, fmt.Errorf("*** Failed to evaluate transaction readAsset: %w", err)
	}

	var asset Asset
	err = json.Unmarshal(r_asset, &asset)
	if err != nil {
		return nil, fmt.Errorf("*** Failed to evaluate transaction readAsset: %w", err)
	}

	return &asset, nil
}

func updateAsset(contract *client.Contract, record Record) error { // Funkcja aktualizująca wartości assetu o danym ID
	_, err := contract.SubmitTransaction("UpdateAsset", fmt.Sprint(record.ID), record.Author, record.Date, record.Message)
	if err != nil {
		return fmt.Errorf("*** Failed to submit transaction updateAsset: %w", err)
	}

	fmt.Println("*** Transaction updateAsset committed successfully")
	return nil
}

func updateDatabase(db *sql.DB, asset Asset) error { // Funkcja dokonująca aktualizacji rekordu SQL o danym ID
	_, err := db.Exec("UPDATE announcement SET author=?, date=?, msg=? WHERE id=?", asset.Author, asset.Date, asset.Message, asset.ID)
	if err != nil {
		return fmt.Errorf("*** Failed to update record in database: %w", err)
	}

	fmt.Println("*** SQL record update committed successfully")
	return nil
}




// Dodatkowe funkcje
func readFirstFile(dirPath string) ([]byte, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return nil, err
	}

	fileNames, err := dir.Readdirnames(1)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(path.Join(dirPath, fileNames[0]))
}

// Format JSON data
func formatJSON(data []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, "", "  "); err != nil {
		panic(fmt.Errorf("*** Failed to parse JSON: %w", err))
	}
	return prettyJSON.String()
}

