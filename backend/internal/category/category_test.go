package category

import "testing"

func TestSuggest(t *testing.T) {
	cases := []struct {
		texts []string
		want  string
	}{
		{[]string{"Immich", "HTTP alt"}, "Media"},
		{[]string{"", "Proxmox VE"}, "Virtualization"},
		{[]string{"FortiSwitch-148F", "", "SSH"}, "Network"},
		{[]string{"ghcr.io/immich-app/immich-server:release"}, "Media"},
		{[]string{"Pi-hole - dns3"}, "Network"},
		{[]string{"lscr.io/linuxserver/sonarr"}, "Downloads"},
		{[]string{"Brother Industries, Ltd."}, "Printers"},
		{[]string{"Private MAC"}, "Phones and tablets"},
		{[]string{"Raspberry Pi Trading Ltd"}, "Computers"},
		{[]string{"postgres:16"}, "Databases"},
		{[]string{"tvheadend"}, ""},
		{[]string{"Living room TV"}, "Media"},
		{[]string{"nothing here"}, ""},
		{[]string{"dns3"}, "Network"},
		{[]string{"hue-bridge"}, "Home automation"},
		{[]string{"lscr.io/linuxserver/home-assistant"}, "Home automation"},
		{[]string{"sshd"}, ""},
	}
	for _, c := range cases {
		if got := Suggest(c.texts...); got != c.want {
			t.Errorf("Suggest(%q) = %q, want %q", c.texts, got, c.want)
		}
	}
}
