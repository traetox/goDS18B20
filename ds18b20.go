package goDS18B20

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
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
	errNoSlaves = errors.New("No temerature probes found")
	errNotFound = errors.New("not found")
	errCRCError = errors.New("CRC error")
	errFormat   = errors.New("Invalid sensor output format")
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
	pbs []*Probe
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
		}
	}
	return slaves, nil
}

func New() (*ProbeGroup, error) {
	var pbs []*Probe
	//get list of potential slaves
	slaves, err := Slaves()
	if err != nil {
		return nil, err
	}
	if len(slaves) == 0 {
		return nil, errNoSlaves
	}
	for i := range slaves {
		pb, err := NewProbe(slaves[i])
		if err != nil {
			return nil, err
		}
		pbs = append(pbs, pb)
	}
	return &ProbeGroup{
		pbs: pbs,
		mtx: &sync.Mutex{},
	}, nil
}

func (pg *ProbeGroup) Close() error {
	pg.mtx.Lock()
	defer pg.mtx.Unlock()
	if pg.pbs == nil {
		return errClosed
	}
	for i := range pg.pbs {
		if err := pg.pbs[i].Close(); err != nil {
			return err
		}
	}
	pg.pbs = nil
	return nil
}

func (pg *ProbeGroup) AssignAlias(alias, id string) error {
	pg.mtx.Lock()
	defer pg.mtx.Unlock()
	if pg.pbs == nil {
		return errClosed
	}
	for i := range pg.pbs {
		if pg.pbs[i].id == id {
			pg.pbs[i].alias = alias
			return nil
		}
	}
	return errNotFound
}

func (pg *ProbeGroup) ReadSingle(id string) (Temperature, error) {
	pg.mtx.Lock()
	defer pg.mtx.Unlock()
	if pg.pbs == nil {
		return -1.0, errClosed
	}
	for i := range pg.pbs {
		if pg.pbs[i].id == id {
			return pg.pbs[i].Temperature()
		}
	}
	return -1.0, errNotFound
}

func (pg *ProbeGroup) ReadSingleAlias(alias string) (Temperature, error) {
	pg.mtx.Lock()
	defer pg.mtx.Unlock()
	if pg.pbs == nil {
		return -1.0, errClosed
	}
	for i := range pg.pbs {
		if pg.pbs[i].alias == alias {
			return pg.pbs[i].Temperature()
		}
	}
	return -1.0, errNotFound
}

func (pg *ProbeGroup) Read() (map[string]Temperature, error) {
	pg.mtx.Lock()
	defer pg.mtx.Unlock()
	if pg.pbs == nil {
		return nil, errClosed
	}
	r := make(map[string]Temperature, 1)
	for i := range pg.pbs {
		t, err := pg.pbs[i].Temperature()
		if err != nil {
			return nil, err
		}
		r[pg.pbs[i].id] = t
	}
	return r, nil
}

func (pg *ProbeGroup) ReadAlias() (map[string]Temperature, error) {
	pg.mtx.Lock()
	defer pg.mtx.Unlock()
	if pg.pbs == nil {
		return nil, errClosed
	}
	r := make(map[string]Temperature, 1)
	for i := range pg.pbs {
		t, err := pg.pbs[i].Temperature()
		if err != nil {
			return nil, err
		}
		if pg.pbs[i].alias == "" {
			continue
		}
		r[pg.pbs[i].alias] = t
	}
	return r, nil
}

func (pg *ProbeGroup) Update() error {
	pg.mtx.Lock()
	defer pg.mtx.Unlock()
	if pg.pbs == nil {
		return errClosed
	}
	for i := range pg.pbs {
		if err := pg.pbs[i].Update(); err != nil {
			return err
		}
	}
	return nil
}

func NewProbe(id string) (*Probe, error) {
	fio, err := os.Open(path.Join(path.Join(basePath, id), "w1_slave"))
	if err != nil {
		return nil, err
	}
	return &Probe{
		currC: 0.0,
		fio:   fio,
		mtx:   &sync.Mutex{},
		id:    id,
	}, nil
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
	bio := bufio.NewReader(p.fio)

	//read first line with CRC
	ln, err := bio.ReadString('\n')
	if err != nil {
		return err
	}
	ln = strings.TrimRight(ln, "\n")

	//pull CRC and check it
	if !strings.HasSuffix(ln, "YES") {
		return errCRCError
	}

	//read second line with temparture
	ln, err = bio.ReadString('\n')
	if err != nil {
		return err
	}
	ln = strings.TrimRight(ln, "\n")
	//pull temparture and convert it
	bits := strings.Split(ln, "t=")
	if len(bits) != 2 {
		return errFormat
	}
	t, err := strconv.ParseUint(bits[1], 10, 32)
	if err != nil {
		return err
	}

	//got a good read, convert it
	temp := float32(t)
	temp = temp / 1000.0
	p.currC = temp
	return nil
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
