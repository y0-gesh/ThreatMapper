package ingesters

import (
	"context"
	"fmt"

	"github.com/deepfence/ThreatMapper/deepfence_utils/directory"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type SecretIngester struct{}

type Secret struct {
	DocId               string `json:"doc_id"`
	Timestamp           string `json:"@timestamp"`
	Count               int    `json:"count,omitempty"`
	Reason              string `json:"reason"`
	Resource            string `json:"resource"`
	Status              string `json:"status"`
	Region              string `json:"region"`
	AccountID           string `json:"account_id"`
	Group               string `json:"group"`
	Service             string `json:"service"`
	Title               string `json:"title"`
	ComplianceCheckType string `json:"compliance_check_type"`
	CloudProvider       string `json:"cloud_provider"`
	NodeName            string `json:"node_name"`
	NodeID              string `json:"node_id"`
	ScanID              string `json:"scan_id"`
	Masked              string `json:"masked"`
	Type                string `json:"type"`
	ControlID           string `json:"control_id"`
	Description         string `json:"description"`
	Severity            string `json:"severity"`
}

type SecretStruct struct {
	Rule     map[string]interface{} `json:"Rule"`
	Severity map[string]interface{} `json:"Severity"`
	Match    map[string]interface{} `json:"Match"`
}

func NewSecretIngester() Ingester[[]map[string]interface{}] {
	return &SecretIngester{}
}

func (tc *SecretIngester) Ingest(ctx context.Context, cs []map[string]interface{}) error {
	driver, err := directory.Neo4jClient(ctx)
	session, err := driver.Session(neo4j.AccessModeWrite)

	if err != nil {
		return err
	}
	defer session.Close()

	tx, err := session.BeginTransaction()
	if err != nil {
		return err
	}
	defer tx.Close()

	secrets := []map[string]interface{}{}
	for _, i := range cs {
		secret := map[string]interface{}{}
		match := i["Match"].(map[string]interface{})
		severity := i["Severity"].(map[string]interface{})
		rule := i["Rule"].(map[string]interface{})

		for k, v := range i {
			if k == "Match" || k == "Severity" || k == "Rule" {
				continue
			}
			secret[k] = v
		}

		for k, v := range rule {
			secret[k] = v
		}
		for k, v := range severity {
			secret[k] = v
		}
		for k, v := range match {
			secret[k] = v
		}
		secret["rule_id"] = fmt.Sprintf("%v:%v", rule["id"], i["host_name"])
		secrets = append(secrets, secret)
	}

	if _, err = tx.Run("UNWIND $batch as row MERGE (n:Secret{node_id:row.rule_id}) MERGE (m:SecretScan{node_id: row.scan_id, host_name: row.host_name, time_stamp: timestamp()}) MERGE (m) -[:DETECTED]-> (n) SET n+= row", map[string]interface{}{"batch": secrets}); err != nil {
		return err
	}

	if _, err = tx.Run("MATCH (n:SecretScan) MERGE (m:Node{node_id: n.host_name}) MERGE (n) -[:SCANNED]-> (m)", map[string]interface{}{}); err != nil {
		return err
	}

	return tx.Commit()
}