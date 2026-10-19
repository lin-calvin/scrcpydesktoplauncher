package reconcile

import "testing"

func TestApplyNamePostfix(t *testing.T) {
	cases := []struct {
		postfix, label, serial, want string
	}{
		{"", "Termux", "ABC123", "Termux"},
		{" ({deviceid})", "Termux", "ABC123", "Termux (ABC123)"},
		{" [{serial}]", "Maps", "10.0.0.2:5555", "Maps [10.0.0.2:5555]"},
		{" ({deviceid})", "A", "X", "A (X)"},
	}
	for _, c := range cases {
		if got := ApplyNamePostfix(c.postfix, c.label, c.serial); got != c.want {
			t.Errorf("ApplyNamePostfix(%q,%q,%q)=%q want %q", c.postfix, c.label, c.serial, got, c.want)
		}
	}
}
