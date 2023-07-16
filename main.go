package main

// This example implements a NUS (Nordic UART Service) client. See nusserver for
// details.

import (
//	"time"
	"fmt"
	"io"
	"os"
	"tinygo.org/x/bluetooth"
	"tinygo.org/x/bluetooth/rawterm"
)

var (
	rescueServiceUUID, err1 = bluetooth.ParseUUID("99EB1511-A9E9-4024-B0A4-3DC4B4FABFB0");
	rescueConfUUID, err2 = bluetooth.ParseUUID("99EB1513-A9E9-4024-B0A4-3DC4B4FABFB0");
	rescueFWUUID, err3 = bluetooth.ParseUUID("99EB1514-A9E9-4024-B0A4-3DC4B4FABFB0");
)

var adapter = bluetooth.DefaultAdapter

func main() {
	// Enable BLE interface.
	err := adapter.Enable()
	if err != nil {
		println("could not enable the BLE stack:", err.Error())
		return
	}

	// The address to connect to. Set during scanning and read afterwards.
	var foundDevice bluetooth.ScanResult

	// Scan for NUS peripheral.
	println("Scanning...")
	err = adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		if !result.AdvertisementPayload.HasServiceUUID(rescueServiceUUID) {
			return
		}
		foundDevice = result

		// Stop the scan.
		err := adapter.StopScan()
		if err != nil {
			// Unlikely, but we can't recover from this.
			println("failed to stop the scan:", err.Error())
		}
	})
	if err != nil {
		println("could not start a scan:", err.Error())
		return
	}

	// Found a device: print this event.
	if name := foundDevice.LocalName(); name == "" {
		print("Connecting to ", foundDevice.Address.String(), "...")
		println()
	} else {
		print("Connecting to ", name, " (", foundDevice.Address.String(), ")...")
		println()
	}

	// Found a NUS peripheral. Connect to it.
	device, err := adapter.Connect(foundDevice.Address, bluetooth.ConnectionParams{})
	if err != nil {
		println("Failed to connect:", err.Error())
		return
	}

	println("Discovering service...")
	services, err := device.DiscoverServices([]bluetooth.UUID{ rescueServiceUUID})
	if err != nil {
		println("Failed to discover the rESCue Service:", err.Error())
		return
	}
	rESCueService := services[0]

	// Get the two characteristics present in the rESCue service.
	chars, err := rESCueService.DiscoverCharacteristics([]bluetooth.UUID{rescueFWUUID, rescueConfUUID})
	if err != nil {
		println("Failed to discover FW characteristics:", err.Error())
		return
	}
    rESCueFW := chars[0]
	rESCueConf := chars[1]

	println("Connected. Exit console using Ctrl-X.")
	rawterm.Configure()
	defer rawterm.Restore()
	var line []byte
	for {
		ch := rawterm.Getchar()
		line = append(line, ch)

		// Send the current line to the central.
		if ch == '\x18' {
			// The user pressed Ctrl-X, exit the program.
			break
		} else if ch == '\n' {

			_, err := rESCueConf.WriteWithoutResponse([]byte("update=start"))
			if err != nil {
				println("could not send:", err.Error())
			}

			//time.Sleep(1 * time.Second)

			// Reset the slice while keeping the buffer in place.
			line = line[:0]

			const BufferSize = 200
			//file, err := os.Open("/home/dgey/projects/rESCue/.pio/build/wemos_d1_mini32/firmware.bin")
			file, err := os.Open("/var/home/dgey/small")
			if err != nil {
				fmt.Println(err)
				return
			}
			defer file.Close()
		
			buffer := make([]byte, BufferSize)
		
			bytesSoFar := 0
			for {
				bytesread, err := file.Read(buffer)
				if err != nil {
					if err != io.EOF {
						fmt.Println(err)
					}		
					break
				}

				// This performs a "write command" aka "write without response".
				_, err = rESCueFW.WriteWithoutResponse(buffer)
				if err != nil {
					println("could not send:", err.Error())
				}
				bytesSoFar += bytesread
			}
			println("bytes sent: ", bytesSoFar, "\n")
		}
	}
}
