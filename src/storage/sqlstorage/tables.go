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
