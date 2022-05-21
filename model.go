package main

import (
	"gorm.io/gorm"
)

type Product struct {
	gorm.Model
	AuctionID  int `gorm:"unique"`
	Name       string
	Location   string
	Price      string
	Bids       string
	EndTime    string
	Link       string
	OnlyPickup bool
}
