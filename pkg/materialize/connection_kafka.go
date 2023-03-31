package materialize

import (
	"fmt"
	"strings"
)

type KafkaBroker struct {
	Broker                string
	TargetGroupPort       int
	AvailabilityZone      string
	PrivateLinkConnection IdentifierSchemaStruct
}

type ConnectionKafkaBuilder struct {
	Connection
	kafkaBrokers        []KafkaBroker
	kafkaProgressTopic  string
	kafkaSSLCa          ValueSecretStruct
	kafkaSSLCert        ValueSecretStruct
	kafkaSSLKey         IdentifierSchemaStruct
	kafkaSASLMechanisms string
	kafkaSASLUsername   ValueSecretStruct
	kafkaSASLPassword   IdentifierSchemaStruct
	kafkaSSHTunnel      IdentifierSchemaStruct
}

func NewConnectionKafkaBuilder(connectionName, schemaName, databaseName string) *ConnectionKafkaBuilder {
	return &ConnectionKafkaBuilder{
		Connection: Connection{connectionName, schemaName, databaseName},
	}
}

func (b *ConnectionKafkaBuilder) KafkaBrokers(kafkaBrokers []KafkaBroker) *ConnectionKafkaBuilder {
	b.kafkaBrokers = kafkaBrokers
	return b
}

func (b *ConnectionKafkaBuilder) KafkaProgressTopic(kafkaProgressTopic string) *ConnectionKafkaBuilder {
	b.kafkaProgressTopic = kafkaProgressTopic
	return b
}

func (b *ConnectionKafkaBuilder) KafkaSSLCa(kafkaSSLCa ValueSecretStruct) *ConnectionKafkaBuilder {
	b.kafkaSSLCa = kafkaSSLCa
	return b
}

func (b *ConnectionKafkaBuilder) KafkaSSLCert(kafkaSSLCert ValueSecretStruct) *ConnectionKafkaBuilder {
	b.kafkaSSLCert = kafkaSSLCert
	return b
}

func (b *ConnectionKafkaBuilder) KafkaSSLKey(kafkaSSLKey IdentifierSchemaStruct) *ConnectionKafkaBuilder {
	b.kafkaSSLKey = kafkaSSLKey
	return b
}

func (b *ConnectionKafkaBuilder) KafkaSASLMechanisms(kafkaSASLMechanisms string) *ConnectionKafkaBuilder {
	b.kafkaSASLMechanisms = kafkaSASLMechanisms
	return b
}

func (b *ConnectionKafkaBuilder) KafkaSASLUsername(kafkaSASLUsername ValueSecretStruct) *ConnectionKafkaBuilder {
	b.kafkaSASLUsername = kafkaSASLUsername
	return b
}

func (b *ConnectionKafkaBuilder) KafkaSASLPassword(kafkaSASLPassword IdentifierSchemaStruct) *ConnectionKafkaBuilder {
	b.kafkaSASLPassword = kafkaSASLPassword
	return b
}

func (b *ConnectionKafkaBuilder) KafkaSSHTunnel(kafkaSSHTunnel IdentifierSchemaStruct) *ConnectionKafkaBuilder {
	b.kafkaSSHTunnel = kafkaSSHTunnel
	return b
}

func (b *ConnectionKafkaBuilder) Create() string {
	q := strings.Builder{}
	q.WriteString(fmt.Sprintf(`CREATE CONNECTION %s TO KAFKA (`, b.QualifiedName()))

	if b.kafkaSSHTunnel.Name != "" {
		q.WriteString(`BROKERS (`)
		for i, broker := range b.kafkaBrokers {
			q.WriteString(fmt.Sprintf(`%s USING SSH TUNNEL %s`, QuoteString(broker.Broker), QualifiedName(b.kafkaSSHTunnel.DatabaseName, b.kafkaSSHTunnel.SchemaName, b.kafkaSSHTunnel.Name)))
			if i < len(b.kafkaBrokers)-1 {
				q.WriteString(`,`)
			}
		}
		q.WriteString(`)`)
	} else {
		q.WriteString(`BROKERS (`)
		for i, broker := range b.kafkaBrokers {
			if broker.TargetGroupPort != 0 && broker.AvailabilityZone != "" && broker.PrivateLinkConnection.Name != "" {
				q.WriteString(fmt.Sprintf(`%s USING AWS PRIVATELINK %s (PORT %d, AVAILABILITY ZONE %s)`, QuoteString(broker.Broker),
					QualifiedName(broker.PrivateLinkConnection.DatabaseName, broker.PrivateLinkConnection.SchemaName, broker.PrivateLinkConnection.Name), broker.TargetGroupPort, QuoteString(broker.AvailabilityZone)))
				if i < len(b.kafkaBrokers)-1 {
					q.WriteString(`, `)
				}
			} else {
				q.WriteString(QuoteString(broker.Broker))
				if i < len(b.kafkaBrokers)-1 {
					q.WriteString(`, `)
				}
			}
		}
		q.WriteString(`)`)
	}

	if b.kafkaProgressTopic != "" {
		q.WriteString(fmt.Sprintf(`, PROGRESS TOPIC %s`, QuoteString(b.kafkaProgressTopic)))
	}
	if b.kafkaSSLCa.Text != "" {
		q.WriteString(fmt.Sprintf(`, SSL CERTIFICATE AUTHORITY = %s`, QuoteString(b.kafkaSSLCa.Text)))
	}
	if b.kafkaSSLCa.Secret.Name != "" {
		q.WriteString(fmt.Sprintf(`, SSL CERTIFICATE AUTHORITY = SECRET %s`, b.kafkaSSLCa.Secret.QualifiedName()))
	}
	if b.kafkaSSLCert.Text != "" {
		q.WriteString(fmt.Sprintf(`, SSL CERTIFICATE = %s`, QuoteString(b.kafkaSSLCert.Text)))
	}
	if b.kafkaSSLCert.Secret.Name != "" {
		q.WriteString(fmt.Sprintf(`, SSL CERTIFICATE = SECRET %s`, b.kafkaSSLCert.Secret.QualifiedName()))
	}
	if b.kafkaSSLKey.Name != "" {
		q.WriteString(fmt.Sprintf(`, SSL KEY = SECRET %s`, b.kafkaSSLKey.QualifiedName()))
	}
	if b.kafkaSASLMechanisms != "" {
		q.WriteString(fmt.Sprintf(`, SASL MECHANISMS = %s`, QuoteString(b.kafkaSASLMechanisms)))
	}
	if b.kafkaSASLUsername.Text != "" {
		q.WriteString(fmt.Sprintf(`, SASL USERNAME = %s`, QuoteString(b.kafkaSASLUsername.Text)))
	}
	if b.kafkaSASLUsername.Secret.Name != "" {
		q.WriteString(fmt.Sprintf(`, SASL USERNAME = SECRET %s`, b.kafkaSASLUsername.Secret.QualifiedName()))
	}
	if b.kafkaSASLPassword.Name != "" {
		q.WriteString(fmt.Sprintf(`, SASL PASSWORD = SECRET %s`, b.kafkaSASLPassword.QualifiedName()))
	}

	q.WriteString(`);`)
	return q.String()
}

func (b *ConnectionKafkaBuilder) Rename(newConnectionName string) string {
	n := QualifiedName(b.DatabaseName, b.SchemaName, newConnectionName)
	return fmt.Sprintf(`ALTER CONNECTION %s RENAME TO %s;`, b.QualifiedName(), n)
}

func (b *ConnectionKafkaBuilder) Drop() string {
	return fmt.Sprintf(`DROP CONNECTION %s;`, b.QualifiedName())
}

func (b *ConnectionKafkaBuilder) ReadId() string {
	return ReadConnectionId(b.ConnectionName, b.SchemaName, b.DatabaseName)
}
