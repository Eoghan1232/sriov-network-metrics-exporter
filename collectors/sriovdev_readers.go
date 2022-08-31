//This file should contain different sriov stat implementations for different drivers and versions.

package collectors

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sriov-network-metrics-exporter/pkg/vfstats"
	"strconv"
	"strings"
)

type sriovStats map[string]int64

//sriovStatReader is an interface which takes in the Physical Function name and vf id and returns the stats for the VF
type sriovStatReader interface {
	ReadStats(vfID string, pfName string) sriovStats
}

//netlinkReader is able to read stats from drivers that support the netlink interface
type netlinkReader struct {
	data vfstats.PerPF
}

//sysfsReader is able to read stats from Physical Functions running the i40e or ice driver.
//other drivers that store all VF stats in files under one folder could use this reader.
type sysfsReader struct {
	statsFS string
}

//statReaderForPF returns the correct stat reader for the given PF
//currently only i40e and ice are implemented, but other drivers can be implemented and picked up here.
func statReaderForPF(pf string, priority []string) sriovStatReader {
	for _, collector := range priority {
		switch collector {
		case "sysfs":
			if *sysfsEnabled {
				if _, err := os.Stat(filepath.Join(*sysClassNet, pf, "/device/sriov")); !os.IsNotExist(err) {
					log.Printf("%s - using sysfs collector", pf)
					return sysfsReader{filepath.Join(*sysClassNet, "%s/device/sriov/%s/stats/")}
				}
				log.Printf("%s does not support sysfs collector", pf)
			}
		case "netlink":
			if *netlinkEnabled {
				log.Printf("%s - using netlink collector", pf)
				return netlinkReader{vfstats.VfStats(pf)}
			}
		default:
			log.Printf("%s - '%s' collector not supported", pf, collector)
			return nil
		}
	}
	return nil
}

//ReadStats takes in the name of a PF and the VF Id and returns a stats object.
func (r netlinkReader) ReadStats(pfName string, vfID string) sriovStats {
	id, err := strconv.Atoi(vfID)
	if err != nil {
		log.Print("Error reading passed Virtual Function ID")
		return sriovStats{}
	}
	return func() sriovStats {
		vf := r.data.Vfs[id]
		return map[string]int64{
			"tx_bytes":     int64(vf.TxBytes),
			"rx_bytes":     int64(vf.RxBytes),
			"tx_packets":   int64(vf.TxPackets),
			"rx_packets":   int64(vf.RxPackets),
			"tx_dropped":   int64(vf.TxDropped),
			"rx_dropped":   int64(vf.RxDropped),
			"rx_broadcast": int64(vf.Broadcast),
			"rx_multicast": int64(vf.Multicast),
		}
	}()
}

func (r sysfsReader) ReadStats(pfName string, vfID string) sriovStats {
	stats := make(sriovStats, 0)
	statroot := fmt.Sprintf(r.statsFS, pfName, vfID)
	files, err := ioutil.ReadDir(statroot)
	if err != nil {
		log.Printf("error reading directory, %v", statroot)
		return stats
	}

	for _, f := range files {
		path := filepath.Join(statroot, f.Name())
		if isSymLink(path) {
			log.Printf("cannot read symlink '%s'", path)
			continue
		}
		statRaw, err := ioutil.ReadFile(path)
		if err != nil {
			log.Printf("error reading file, %v", err)
			continue
		}
		statString := strings.TrimSpace(string(statRaw))
		value, err := strconv.ParseInt(statString, 10, 64)
		if err != nil {
			log.Printf("error parsing file '%v', error: %v", f.Name(), err)
			continue
		}
		stats[f.Name()] = value
	}
	return stats
}
