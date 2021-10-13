package schedule

import (
	"bytes"
	"encoding/gob"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/pkg/errors"
)

const (
	dynamoIDKey       = "ChannelId"
	dynamoScheduleKey = "Schedule"
)

type DynamoDBRepository struct {
	dynamodbCli *dynamodb.DynamoDB
	tableName   string
}

func NewDynamoDBRepostory(sess *session.Session, tableName string) *DynamoDBRepository {
	dynDbCli := dynamodb.New(sess)

	return &DynamoDBRepository{
		dynamodbCli: dynDbCli,
		tableName:   tableName,
	}
}

func (r *DynamoDBRepository) Get(id string) (*Schedule, error) {
	resp, err := r.dynamodbCli.GetItem(&dynamodb.GetItemInput{
		TableName: &r.tableName,
		Key: map[string]*dynamodb.AttributeValue{
			dynamoIDKey: {
				S: aws.String(id),
			},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return nil, errors.Wrap(err, "while getting item")
	}

	value := resp.Item[dynamoScheduleKey]
	if value == nil {
		return &Schedule{}, nil
	}

	dec := gob.NewDecoder(bytes.NewBuffer(value.B))

	list := &Schedule{}
	if err := dec.Decode(list); err != nil {
		return nil, errors.Wrap(err, "while decoding schedule value")
	}

	return list, nil
}

func (r *DynamoDBRepository) Save(id string, l *Schedule) error {
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)

	if err := enc.Encode(l); err != nil {
		return errors.Wrap(err, "while encoding schedule value")
	}

	_, err := r.dynamodbCli.PutItem(&dynamodb.PutItemInput{
		TableName: &r.tableName,
		Item: map[string]*dynamodb.AttributeValue{
			dynamoIDKey: {
				S: aws.String(id),
			},
			dynamoScheduleKey: {
				B: buf.Bytes(),
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "while putting item")
	}

	return nil
}
