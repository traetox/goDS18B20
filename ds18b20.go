package ds18b20

import (
	"os"
	"path"
	"sync"
	"strings"
	"strconv"
)

const (
	basePath = `/sys/bus/w1/devices/`
)

var (
	errClose = errors.New("Closed")
)

type Temperature float32

type Probe struct {
	currC float32
	fio *os.File
	mtx *sync.Mutex
	id string
	alias string
}

type ProbeGroup struct {
	pbs []Probe
	mtx *sync.Mutex
}

//Setup will ensure the w1-gpio and w1-therm modules are loaded
//and ensure that there is at least one w1_bus_masterX present
func Setup() error {

}

//Slaves will return a listing of available slaves
func Slaves() ([]string, error) {
	
}

func New() (*ProbeGroup, error) {

}

func (pg *ProbeGroup) Close() error {

}

func (pg *ProbeGroup) ReadSingle(id string) (Temperature, error) {

}

func (pg *ProbeGroup) ReadSingleAlias(alias string) (Temperature, error) {

}

func (pg *ProbeGroup) Read() (map[string]Temperature, error) {

}

func (pg *ProbeGroup) ReadAlias() (map[string]Temperature, error) {

}

func (p *Probe) NewProbe(alias string) (*Probe, error) {

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

func (t Temparature) Celsius() float32 {
	return t
}

func (t Temparature) Fahrenheit() float32 {
	return (t*1.8) + 32.0
}

func (t Temparature) Kelvin() float32 {
	return t-273.15
}

func (t Temparature) Centigrade() float32 {
	return t.Kelvin()
}

func (t Temparature) String() string {
	return fmt.Sprintf("%.03f C", t)
}
