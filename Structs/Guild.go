package Structs

import (
	"github.com/disgoorg/snowflake/v2"
)

type Guild struct {

	ID snowflake.ID `json:"id"`
	Name string `json:"name"`

	Queue Queue `json:"queue"`

	Channels Channels `json:"channels"`
	
	Features Features `json:"features"`

}

type Queue struct {

	Previous []Song `json:"previous"`
	Current Song `json:"current"`
	Next []Song `json:"next"`

}

type Channels struct {

	Voice snowflake.ID `json:"voice"`
	Text snowflake.ID `json:"text"`

}

const (

	RepeatOff = iota
	RepeatOne = iota
	RepeatAll = iota

)

type Features struct {

	Repeat int `json:"repeat"`
	Shuffle bool `json:"shuffle"`
	Autoplay bool `json:"autoplay"`

}