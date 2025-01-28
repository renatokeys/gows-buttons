package sqlstorage

type Table struct {
	Name             string
	Columns          []string
	DataField        string
	OnConflict       []string
	UpdateOnConflict []string
}

var MessageTable = Table{
	Name: "gows_messages",
	Columns: []string{
		"jid",
		"id",
		"timestamp",
		"from_me",
		"data",
	},
	DataField: "data",
	OnConflict: []string{
		"id",
	},
	UpdateOnConflict: []string{
		"timestamp",
		"data",
	},
}

var ChatTable = Table{
	Name: "gows_chats",
	Columns: []string{
		"jid",
		"name",
		"conversation_timestamp",
		"data",
	},
	DataField: "data",
	OnConflict: []string{
		"jid",
	},
	UpdateOnConflict: []string{
		"name",
		"conversation_timestamp",
		"data",
	},
}
