package lineup

import "testing"

func TestShortName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "Josh Allen", want: "J Allen"},
		{name: "Mahomes", want: "Mahomes"},
		{name: "", want: ""},
		{name: "Émile Smith", want: "É Smith"},
		{name: "Will McDonald IV", want: "W McDonald"},
		{name: "Marvin Harrison Jr.", want: "M Harrison"},
		{name: "Marvin Harrison Jr", want: "M Harrison"},
		{name: "Luther Burden III", want: "L Burden"},
		{name: "Ken Griffey Sr.", want: "K Griffey"},
		{name: "Ken Griffey Sr", want: "K Griffey"},
		{name: "Robert Griffin II", want: "R Griffin"},
		{name: "Frank Smith V", want: "F Smith"},
		{name: "Tony V", want: "T V"},
		{name: "Amon-Ra St. Brown", want: "A St. Brown"},
		{name: "Ja'Marr Chase", want: "J Chase"},
		{name: "Josh  Allen", want: "J Allen"},
		{name: "A.J. Brown", want: "A.J. Brown"},
		{name: "KC Concepcion", want: "KC Concepcion"},
		{name: "DJ Moore", want: "DJ Moore"},
		{name: "DK Metcalf", want: "DK Metcalf"},
		{name: "TJ Watt", want: "TJ Watt"},
		{name: "JK Dobbins", want: "JK Dobbins"},
		{name: "Al Smith", want: "A Smith"},
		{name: "JOE Smith", want: "J Smith"},
		{name: "CeeDee Lamb", want: "C Lamb"},
		{name: "A.J. Brown Jr.", want: "A.J. Brown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shortName(tt.name); got != tt.want {
				t.Errorf("shortName(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}
