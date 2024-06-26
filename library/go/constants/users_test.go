package constants

import (
	"reflect"
	"testing"
)

func TestSlackUsers(t *testing.T) {
	expected := &Users{
		PaulWaltersDev:     "U020E8T5PC5",
		MeganSitoyPractera: "U04MFRK6350",
		Rodentskie:         "U06Q5GKADME",
		Trtshen:            "U1GBY5XKJ",
		TerenceCoder:       "U1GJ48N5V",
		Jazzmind:           "U02TB2WV7",
		Shawnm0705:         "U0DFSQLJC",
		Sasangachathumal:   "U9JD7Q9GF",
		Sunilsbcloud:       "U016R04BE81",
		Rodentskiie:        "U06Q7E7QFNX",
	}

	result := SlackUsers()

	if !reflect.DeepEqual(expected, result) {
		t.Errorf("struct does not match the expected.")
	}
}
