package commands

import (
	"crypto/sha512"
	"fmt"
	"github.com/spf13/cobra"
	"net"
)

// ShowMacHashesCmd dumps the MAC addresses hashes.
var ShowMacHashesCmd = &cobra.Command{
	Use:   "show_mac_hashes",
	Short: "Show this node's MAC hashes",
	RunE:  showMacHashes,
}

func getMacAddr() ([]string, error) {
	netInts, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var macs []string
	for _, netInt := range netInts {
		mac := netInt.HardwareAddr.String()
		if mac != "" {
			macs = append(macs, mac)
		}
	}
	return macs, nil
}

func calcSHA512Hash(input string) string {
	h := sha512.New()
	h.Write([]byte(input))
	macHash := h.Sum(nil)
	return fmt.Sprintf("%x", macHash)
}

func showMacHashes(cmd *cobra.Command, args []string) error {

	macs, _ := getMacAddr()
	for i, mac := range macs {
		fmt.Printf("INTERFACE n.%d: MAC=%s -> HASH(SHA512)=%s\n", i+1, mac, calcSHA512Hash(mac))
	}
	return nil
}
