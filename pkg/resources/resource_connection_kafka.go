package resources

import (
	"context"
	"log"
	"strings"

	"github.com/MaterializeInc/terraform-provider-materialize/pkg/materialize"
	"github.com/MaterializeInc/terraform-provider-materialize/pkg/utils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var connectionKafkaSchema = map[string]*schema.Schema{
	"name":               ObjectNameSchema("connection", true, false),
	"schema_name":        SchemaNameSchema("connection", false),
	"database_name":      DatabaseNameSchema("connection", false),
	"qualified_sql_name": QualifiedNameSchema("connection"),
	"comment":            CommentSchema(false),
	"kafka_broker": {
		Description:   "The Kafka broker's configuration.",
		Type:          schema.TypeList,
		ConflictsWith: []string{"aws_privatelink"},
		AtLeastOneOf:  []string{"kafka_broker", "aws_privatelink"},
		Optional:      true,
		MinItems:      1,
		ForceNew:      true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"broker": {
					Description: "The Kafka broker, in the form of `host:port`.",
					Type:        schema.TypeString,
					Required:    true,
				},
				"target_group_port": {
					Description: "The port of the target group associated with the Kafka broker.",
					Type:        schema.TypeInt,
					Optional:    true,
				},
				"availability_zone": {
					Description: "The availability zone of the Kafka broker.",
					Type:        schema.TypeString,
					Optional:    true,
				},
				"privatelink_connection": IdentifierSchema(IdentifierSchemaParams{
					Elem:        "privatelink_connection",
					Description: "The AWS PrivateLink connection name in Materialize.",
					Required:    false,
					ForceNew:    true,
				}),
				"ssh_tunnel": IdentifierSchema(IdentifierSchemaParams{
					Elem:        "ssh_tunnel",
					Description: "The name of an SSH tunnel connection to route network traffic through by default.",
					Required:    false,
					ForceNew:    true,
				}),
			},
		},
	},
	"aws_privatelink": {
		Description:   "AWS PrivateLink configuration. Conflicts with `kafka_broker`.",
		Type:          schema.TypeList,
		Optional:      true,
		ConflictsWith: []string{"kafka_broker"},
		AtLeastOneOf:  []string{"kafka_broker", "aws_privatelink"},
		MinItems:      1,
		MaxItems:      1,
		ForceNew:      true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"privatelink_connection": IdentifierSchema(IdentifierSchemaParams{
					Elem:        "privatelink_connection",
					Description: "The AWS PrivateLink connection name in Materialize.",
					Required:    true,
					ForceNew:    true,
				}),
				"privatelink_connection_port": {
					Description: "The port of the AWS PrivateLink connection.",
					Type:        schema.TypeInt,
					Required:    true,
					ForceNew:    true,
				},
			},
		},
	},
	"security_protocol": {
		Description:  "The security protocol to use: `PLAINTEXT`, `SSL`, `SASL_PLAINTEXT`, or `SASL_SSL`.",
		Type:         schema.TypeString,
		Optional:     true,
		ForceNew:     true,
		ValidateFunc: validation.StringInSlice(securityProtocols, true),
		StateFunc: func(val any) string {
			return strings.ToUpper(val.(string))
		},
	},
	"progress_topic": {
		Description: "The name of a topic that Kafka sinks can use to track internal consistency metadata.",
		Type:        schema.TypeString,
		Optional:    true,
		ForceNew:    true,
	},
	"ssl_certificate_authority": ValueSecretSchema("ssl_certificate_authority", "The CA certificate for the Kafka broker.", false, true),
	"ssl_certificate":           ValueSecretSchema("ssl_certificate", "The client certificate for the Kafka broker.", false, true),
	"ssl_key": IdentifierSchema(IdentifierSchemaParams{
		Elem:        "ssl_key",
		Description: "The client key for the Kafka broker.",
		Required:    false,
		ForceNew:    true,
	}),
	"sasl_mechanisms": {
		Description:  "The SASL mechanism for the Kafka broker.",
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: validation.StringInSlice(saslMechanisms, true),
		RequiredWith: []string{"sasl_username", "sasl_password"},
		StateFunc: func(val any) string {
			return strings.ToUpper(val.(string))
		},
		ForceNew: true,
	},
	"sasl_username": ValueSecretSchema("sasl_username", "The SASL username for the Kafka broker.", false, true),
	"sasl_password": IdentifierSchema(IdentifierSchemaParams{
		Elem:        "sasl_password",
		Description: "The SASL password for the Kafka broker.",
		Required:    false,
		ForceNew:    true,
	}),
	"ssh_tunnel": IdentifierSchema(IdentifierSchemaParams{
		Elem:        "ssh_tunnel",
		Description: "The default SSH tunnel configuration for the Kafka brokers.",
		Required:    false,
		ForceNew:    true,
	}),
	"validate":       ValidateConnectionSchema(),
	"ownership_role": OwnershipRoleSchema(),
	"region":         RegionSchema(),
}

func ConnectionKafka() *schema.Resource {
	return &schema.Resource{
		Description: "A Kafka connection establishes a link to a Kafka cluster.",

		CreateContext: connectionKafkaCreate,
		ReadContext:   connectionRead,
		UpdateContext: connectionUpdate,
		DeleteContext: connectionDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: connectionKafkaSchema,
	}
}

func connectionKafkaCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	connectionName := d.Get("name").(string)
	schemaName := d.Get("schema_name").(string)
	databaseName := d.Get("database_name").(string)

	metaDb, region, err := utils.GetDBClientFromMeta(meta, d)
	if err != nil {
		return diag.FromErr(err)
	}
	o := materialize.MaterializeObject{ObjectType: "CONNECTION", Name: connectionName, SchemaName: schemaName, DatabaseName: databaseName}
	b := materialize.NewConnectionKafkaBuilder(metaDb, o)

	if v, ok := d.GetOk("kafka_broker"); ok {
		brokers := materialize.GetKafkaBrokersStruct(v)
		b.KafkaBrokers(brokers)
	}

	if v, ok := d.GetOk("aws_privatelink"); ok {
		privatelink := materialize.GetAwsPrivateLinkConnectionStruct(v)
		b.KafkaAwsPrivateLink(privatelink)
	}

	if v, ok := d.GetOk("security_protocol"); ok {
		b.KafkaSecurityProtocol(v.(string))
	}

	if v, ok := d.GetOk("progress_topic"); ok {
		b.KafkaProgressTopic(v.(string))
	}

	if v, ok := d.GetOk("ssl_certificate_authority"); ok {
		ssl_ca := materialize.GetValueSecretStruct(v)
		b.KafkaSSLCa(ssl_ca)
	}

	if v, ok := d.GetOk("ssl_certificate"); ok {
		ssl_cert := materialize.GetValueSecretStruct(v)
		b.KafkaSSLCert(ssl_cert)
	}

	if v, ok := d.GetOk("ssl_key"); ok {
		key := materialize.GetIdentifierSchemaStruct(v)
		b.KafkaSSLKey(key)
	}

	if v, ok := d.GetOk("sasl_mechanisms"); ok {
		b.KafkaSASLMechanisms(v.(string))
	}

	if v, ok := d.GetOk("sasl_username"); ok {
		sasl_username := materialize.GetValueSecretStruct(v)
		b.KafkaSASLUsername(sasl_username)
	}

	if v, ok := d.GetOk("sasl_password"); ok {
		pass := materialize.GetIdentifierSchemaStruct(v)
		b.KafkaSASLPassword(pass)
	}

	if v, ok := d.GetOk("ssh_tunnel"); ok {
		conn := materialize.GetIdentifierSchemaStruct(v)
		b.KafkaSSHTunnel(conn)
	}

	if v, ok := d.GetOk("validate"); ok {
		b.Validate(v.(bool))
	}

	// create resource
	if err := b.Create(); err != nil {
		return diag.FromErr(err)
	}

	// ownership
	if v, ok := d.GetOk("ownership_role"); ok {
		ownership := materialize.NewOwnershipBuilder(metaDb, o)

		if err := ownership.Alter(v.(string)); err != nil {
			log.Printf("[DEBUG] resource failed ownership, dropping object: %s", o.Name)
			b.Drop()
			return diag.FromErr(err)
		}
	}

	// object comment
	if v, ok := d.GetOk("comment"); ok {
		comment := materialize.NewCommentBuilder(metaDb, o)

		if err := comment.Object(v.(string)); err != nil {
			log.Printf("[DEBUG] resource failed comment, dropping object: %s", o.Name)
			b.Drop()
			return diag.FromErr(err)
		}
	}

	// set id
	i, err := materialize.ConnectionId(metaDb, o)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(utils.TransformIdWithRegion(string(region), i))

	return connectionRead(ctx, d, meta)
}
