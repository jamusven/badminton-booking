package data

import (
	"gorm.io/gorm"
)

func init() {
	if err := DBGet().AutoMigrate(&Transaction{}); err != nil {
		panic(err)
	}
}

type Transaction struct {
	gorm.Model
	UID           uint            `gorm:"index"`
	Operator      uint            `gorm:"index"`
	Type          TransactionType `gorm:"index"`
	VenueID       uint            `gorm:"index"`
	ChangeAmount  float32
	CurrentAmount float32
	Desc          string
}

type TransactionType int

const (
	TransactionTypeVenue TransactionType = iota + 1
	TransactionTypeBall
	TransactionTypeTraining
	TransactionTypeBalance
	TransactionTypeFare
)

var TransactionTypeMap = map[TransactionType]string{
	TransactionTypeVenue:    "场地费",
	TransactionTypeBall:     "球费",
	TransactionTypeTraining: "训练费",
	TransactionTypeBalance:  "余额",
	TransactionTypeFare:     "车费",
}

func CreateTransaction(operator, uid, venueId uint, transactionType TransactionType, changeAmount, currentAmount float32, desc string) error {
	tx := DBGet().Create(&Transaction{
		Operator:      operator,
		UID:           uid,
		VenueID:       venueId,
		Type:          transactionType,
		ChangeAmount:  changeAmount,
		CurrentAmount: currentAmount,
		Desc:          desc,
	})

	return tx.Error
}
