// Package audio слушает то, что играет на компьютере, и раскладывает звук
// на громкость и полосы спектра.
//
// Своего звукового стека здесь нет: поток забирается у PipeWire или
// PulseAudio через их же утилиту записи. Это единственная часть программы,
// которой нужна посторонняя команда, и без неё всё остальное работает —
// просто звуковые эффекты будут недоступны.
package audio

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os/exec"
	"strings"
	"sync"
)

const (
	// Частота намеренно невысокая: для подсветки хватает полосы до 11 кГц,
	// а меньше данных — меньше работы на каждый кадр.
	sampleRate = 22050
	blockSize  = 1024 // 46 мс — примерно три кадра при 60 к/с

	// BandCount — сколько полос спектра отдаётся эффектам.
	BandCount = 12
)

// bandHz — центры полос, примерно треть октавы друг от друга.
var bandHz = [BandCount]float64{
	60, 100, 160, 250, 400, 630, 1000, 1600, 2500, 4000, 6300, 9000,
}

// Frame — состояние звука на текущий момент. Всё в диапазоне 0..1.
type Frame struct {
	Level float64
	// Beat — насколько бас сейчас громче своего обычного уровня.
	//
	// Отдельно от Level, потому что громкость сама по себе для ритма не
	// годится: у ровной музыки она почти всё время держится у потолка, и
	// эффект, привязанный к ней, выглядит застывшим. Beat смотрит не на
	// абсолютную величину, а на превышение над скользящим средним, поэтому
	// удары читаются и на тихой, и на громкой записи.
	Beat  float64
	Bands [BandCount]float64
}

// Source — то, что можно слушать.
type Source struct {
	Name    string `json:"name"`
	Label   string `json:"label"`
	Monitor bool   `json:"monitor"`
}

// Capture держит запущенную утилиту записи и считает по её потоку спектр.
type Capture struct {
	mu    sync.Mutex
	frame Frame
	err   error

	cmd  *exec.Cmd
	stop chan struct{}
	done chan struct{}

	// Автоусиление: тихую музыку надо поднимать, громкую прижимать, иначе
	// подсветка либо еле тлеет, либо всё время упирается в максимум.
	peak      float64
	bandPeaks [BandCount]float64
	bassAvg   float64 // обычный уровень баса, относительно него ищем удары
}

// Available сообщает, есть ли чем захватывать звук.
func Available() bool { _, _, err := recorder(); return err == nil }

// recorder выбирает утилиту записи. parec отдаёт сырой поток и потому
// предпочтительнее; pw-record пишет WAV, у него приходится снимать заголовок.
func recorder() (name string, wav bool, err error) {
	if p, e := exec.LookPath("parec"); e == nil {
		return p, false, nil
	}
	if p, e := exec.LookPath("pw-record"); e == nil {
		return p, true, nil
	}
	return "", false, errors.New(
		"нужна утилита записи: parec (pulseaudio-utils) или pw-record (pipewire)")
}

// Sources перечисляет источники звука. Мониторы идут первыми: обычно нужен
// именно системный звук, а не микрофон.
func Sources() ([]Source, error) {
	pactl, err := exec.LookPath("pactl")
	if err != nil {
		return nil, errors.New("не найдена команда pactl")
	}
	out, err := exec.Command(pactl, "list", "short", "sources").Output()
	if err != nil {
		return nil, err
	}
	var mons, ins []Source
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		s := Source{Name: f[1], Label: prettyName(f[1]),
			Monitor: strings.HasSuffix(f[1], ".monitor")}
		if s.Monitor {
			mons = append(mons, s)
		} else {
			ins = append(ins, s)
		}
	}
	return append(mons, ins...), nil
}

// prettyName делает из имени узла что-то читаемое.
func prettyName(name string) string {
	s := strings.TrimSuffix(name, ".monitor")
	if i := strings.Index(s, "."); i >= 0 && i+1 < len(s) {
		s = s[i+1:]
	}
	s = strings.NewReplacer("_", " ", "-", " ", ".", " ").Replace(s)
	if fields := strings.Fields(s); len(fields) > 0 {
		s = strings.Join(fields, " ")
	}
	if strings.HasSuffix(name, ".monitor") {
		return s + " (вывод)"
	}
	return s
}

// DefaultSource — монитор текущего устройства вывода, то есть всё, что
// слышно из колонок.
func DefaultSource() string {
	pactl, err := exec.LookPath("pactl")
	if err != nil {
		return ""
	}
	out, err := exec.Command(pactl, "get-default-sink").Output()
	if err != nil {
		return ""
	}
	if sink := strings.TrimSpace(string(out)); sink != "" {
		return sink + ".monitor"
	}
	return ""
}

// Start запускает захват. Пустой source означает устройство по умолчанию.
func Start(source string) (*Capture, error) {
	bin, wav, err := recorder()
	if err != nil {
		return nil, err
	}
	if source == "" {
		source = DefaultSource()
	}

	var cmd *exec.Cmd
	if wav {
		cmd = exec.Command(bin,
			fmt.Sprintf("--rate=%d", sampleRate), "--channels=1", "--format=s16",
			"--target="+source, "-")
	} else {
		cmd = exec.Command(bin, "--format=s16le",
			fmt.Sprintf("--rate=%d", sampleRate), "--channels=1",
			"--latency-msec=25", "-d", source)
	}

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("не удалось запустить %s: %w", bin, err)
	}

	c := &Capture{cmd: cmd, stop: make(chan struct{}), done: make(chan struct{})}
	go c.read(pipe, wav)
	return c, nil
}

// read крутится, пока утилита отдаёт звук.
func (c *Capture) read(pipe io.ReadCloser, wav bool) {
	defer close(c.done)
	defer pipe.Close()

	r := bufio.NewReaderSize(pipe, blockSize*4)
	if wav {
		// Заголовок WAV нас не интересует, дальше идут те же сэмплы.
		if _, err := io.CopyN(io.Discard, r, 44); err != nil {
			c.setErr(err)
			return
		}
	}

	raw := make([]byte, blockSize*2)
	samples := make([]float64, blockSize)
	for {
		select {
		case <-c.stop:
			return
		default:
		}
		if _, err := io.ReadFull(r, raw); err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrClosedPipe) {
				c.setErr(err)
			}
			return
		}
		for i := range samples {
			samples[i] = float64(int16(binary.LittleEndian.Uint16(raw[i*2:]))) / 32768
		}
		c.process(samples)
	}
}

// process считает громкость и полосы по одному блоку.
func (c *Capture) process(samples []float64) {
	var sum float64
	for _, v := range samples {
		sum += v * v
	}
	rms := math.Sqrt(sum / float64(len(samples)))

	// Окно Ханна: без него на границах блока возникает щелчок, который
	// размазывается по всему спектру и полосы пляшут на ровном звуке.
	windowed := make([]float64, len(samples))
	for i, v := range samples {
		w := 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(len(samples)-1))
		windowed[i] = v * w
	}

	var bands [BandCount]float64
	for i, hz := range bandHz {
		bands[i] = goertzel(windowed, hz)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.peak = decayPeak(c.peak, rms)
	c.frame.Level = smooth(c.frame.Level, norm(rms, c.peak))
	for i := range bands {
		c.bandPeaks[i] = decayPeak(c.bandPeaks[i], bands[i])
		c.frame.Bands[i] = smooth(c.frame.Bands[i], norm(bands[i], c.bandPeaks[i]))
	}

	// Удар ищем по трём нижним полосам и по сырым величинам: нормированные
	// уже подтянуты автоусилением к потолку, и превышение по ним не видно.
	bass := (bands[0] + bands[1] + bands[2]) / 3
	if c.bassAvg == 0 {
		c.bassAvg = bass
	}
	c.bassAvg += (bass - c.bassAvg) * 0.04
	beat := 0.0
	if c.bassAvg > 1e-6 {
		// Во сколько раз бас громче обычного. Полуторное превышение — уже
		// отчётливый удар, на нём и выводим в максимум.
		beat = math.Min(1, math.Max(0, (bass/c.bassAvg-1)/0.5))
	}
	// Спад делаем быстрее, чем у громкости: иначе удары сливаются в свечение.
	if beat > c.frame.Beat {
		c.frame.Beat += (beat - c.frame.Beat) * 0.75
	} else {
		c.frame.Beat += (beat - c.frame.Beat) * 0.28
	}
}

// goertzel считает амплитуду одной частоты. Полос всего дюжина, ради них
// разворачивать полное преобразование Фурье незачем.
func goertzel(x []float64, hz float64) float64 {
	n := float64(len(x))
	k := math.Round(hz * n / sampleRate)
	w := 2 * math.Pi * k / n
	coeff := 2 * math.Cos(w)

	var s0, s1, s2 float64
	for _, v := range x {
		s0 = v + coeff*s1 - s2
		s2, s1 = s1, s0
	}
	power := s1*s1 + s2*s2 - coeff*s1*s2
	if power < 0 {
		power = 0
	}
	return math.Sqrt(power) / n * 2
}

// decayPeak ведёт скользящий максимум: мгновенно поднимается и медленно
// оседает, чтобы после громкого места тихое не выглядело мёртвым.
func decayPeak(peak, v float64) float64 {
	const floor = 0.004 // ниже этого — тишина, поднимать её не нужно
	peak *= 0.995
	if v > peak {
		peak = v
	}
	if peak < floor {
		peak = floor
	}
	return peak
}

// norm приводит значение к 0..1 по логарифмической шкале: ухо слышит
// громкость логарифмически, и линейная картинка выглядит вялой.
func norm(v, peak float64) float64 {
	if v <= 0 || peak <= 0 {
		return 0
	}
	const rangeDB = 28
	db := 20 * math.Log10(v/peak)
	f := 1 + db/rangeDB
	return math.Max(0, math.Min(1, f))
}

// smooth сглаживает: вверх почти мгновенно, вниз плавно — так удары
// читаются, а картинка не дёргается.
func smooth(prev, now float64) float64 {
	if now > prev {
		return prev + (now-prev)*0.6
	}
	return prev + (now-prev)*0.18
}

// Frame отдаёт последнее посчитанное состояние.
func (c *Capture) Frame() Frame {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.frame
}

// Err возвращает ошибку, если захват прервался.
func (c *Capture) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

func (c *Capture) setErr(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.err = err
}

// Stop останавливает захват и дожидается завершения утилиты.
func (c *Capture) Stop() {
	select {
	case <-c.stop:
		return // уже остановлен
	default:
		close(c.stop)
	}
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
	}
	<-c.done
	if c.cmd != nil {
		c.cmd.Wait()
	}
}
