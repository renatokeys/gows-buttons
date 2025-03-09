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
		"is_real",
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

var GroupTable = Table{
	Name: "gows_groups",
	Columns: []string{
		"id",
		"name",
		"data",
	},
	DataField: "data",
	OnConflict: []string{
		"id",
	},
	UpdateOnConflict: []string{
		"name",
		"data",
	},
}
