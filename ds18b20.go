package ds18b20

import (
	"errors"
	"fmt"
	"os"
	//"path"
	"strings"
	"sync"
	//"strconv"
	"io/ioutil"
	"os/exec"
)

const (
	basePath     = `/sys/bus/w1/devices/`
	modprobeCmd  = `/sbin/modprobe`
	w1thermMod   = `w1-therm`
	w1gpioMod    = `w1-gpio`
	masterPrefix = `w1_bus_master`
	slavePrefix  = `28-`
)

var (
	errClosed   = errors.New("Closed")
	errNoBus    = errors.New("1 Wire master bus not present")
	errNotReady = errors.New("Not ready")
)

type Temperature float32

type Probe struct {
	currC float32
	fio   *os.File
	mtx   *sync.Mutex
	id    string
	alias string
}

type ProbeGroup struct {
	pbs []Probe
	mtx *sync.Mutex
}

//Setup will ensure the w1-gpio and w1-therm modules are loaded
//and ensure that there is at least one w1_bus_masterX present
func Setup() error {
	//ensure the modules are loaded
	cmd := exec.Command(modprobeCmd, w1gpioMod)
	if err := cmd.Run(); err != nil {
		return err
	}
	cmd = exec.Command(modprobeCmd, w1thermMod)
	if err := cmd.Run(); err != nil {
		return err
	}

	//check that the 1 wire master device is present
	fis, err := ioutil.ReadDir(basePath)
	if err != nil {
		return err
	}
	masterPresent := false
	for i := range fis {
		if strings.HasPrefix(fis[i].Name(), masterPrefix) {
			masterPresent = true
			break
		}
	}
	if !masterPresent {
		return errNoBus
	}
	//good to go
	return nil
}

//Slaves will return a listing of available slaves
func Slaves() ([]string, error) {
	var slaves []string
	fis, err := ioutil.ReadDir(basePath)
	if err != nil {
		return nil, err
	}
	for i := range fis {
		if strings.HasPrefix(fis[i].Name(), slavePrefix) {
			if (fis[i].Mode() & os.ModeSymlink) == os.ModeSymlink {
				slaves = append(slaves, fis[i].Name())
			}
		} else {
			fmt.Printf("Not a slave: %v\n", fis[i].Name())
		}
	}
	return slaves, nil
}

func New() (*ProbeGroup, error) {
	return nil, errNotReady
}

func (pg *ProbeGroup) Close() error {
	return errNotReady
}

func (pg *ProbeGroup) ReadSingle(id string) (Temperature, error) {
	return -1.0, errNotReady
}

func (pg *ProbeGroup) ReadSingleAlias(alias string) (Temperature, error) {
	return -1.0, errNotReady
}

func (pg *ProbeGroup) Read() (map[string]Temperature, error) {
	return nil, errNotReady

}

func (pg *ProbeGroup) ReadAlias() (map[string]Temperature, error) {
	return nil, errNotReady

}

func (p *Probe) NewProbe(alias string) (*Probe, error) {
	return nil, errNotReady
}

func (p *Probe) Temperature() (Temperature, error) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	if p.fio == nil {
		return 0.0, errClosed
	}
	return Temperature(p.currC), nil
}

func (p *Probe) Update() error {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	if p.fio == nil {
		return errClosed
	}
	if _, err := p.fio.Seek(0, 0); err != nil {
		return err
	}

	//read first line with CRC
	//pull CRC and check it
	//read second line with temparture
	//pull temparture and convert it

	return errNotReady
}

func (p *Probe) Close() error {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	if p.fio == nil {
		return errClosed
	}

	p.fio.Close()
	p.fio = nil
	return nil
}

func (t Temperature) Celsius() float32 {
	return float32(t)
}

func (t Temperature) Fahrenheit() float32 {
	return float32((t * 1.8) + 32.0)
}

func (t Temperature) Kelvin() float32 {
	return float32(t - 273.15)
}

func (t Temperature) Centigrade() float32 {
	return float32(t.Kelvin())
}

func (t Temperature) String() string {
	return fmt.Sprintf("%.03f C", t)
}
