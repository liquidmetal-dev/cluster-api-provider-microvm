package v1alpha1

import (
	"github.com/yitsushi/macpot"
)

func SetDefaults_NetworkInterface(obj *NetworkInterface) { //nolint: revive,stylecheck // idk it was here
	if obj.GuestMAC == "" {
		mac, _ := macpot.New(macpot.AsLocal(), macpot.AsUnicast())

		obj.GuestMAC = mac.ToString()
	}
}
