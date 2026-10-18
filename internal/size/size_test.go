package size

import "testing"

func TestHuman(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1024 * 1024, "1.00 MB"},
		{1536 * 1024, "1.50 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{100 * 1024 * 1024, "100 MB"},
		{1024 * 1024 * 1024 * 1024, "1.00 TB"},
	}
	for _, c := range cases {
		if got := Human(c.in); got != c.want {
			t.Errorf("Human(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestHumanNegative(t *testing.T) {
	if got := Human(-2048); got != "-2.00 KB" {
		t.Errorf("Human(-2048) = %q", got)
	}
}
