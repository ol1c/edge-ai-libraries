# Raport testow NPU: Meteor Lake, 2026-11-13

## Werdykt

Host jest **Meteor Lake (MTL), nie Panther Lake (PTL)**. Potwierdzaja to
CPU family 6/model 170 (0xAA), PCI NPU `8086:7d1d` i PMT GUID `0x130670b2`.
Nazwa hosta nie byla kryterium identyfikacji.

Fizyczne inferencje NPU i odczyty sysfs przeszly krotki idle/load/recovery.
**Fizyczna walidacja metryk kolektora jest zablokowana przez uprawnienia PMT.**
Nie mozna z tego przebiegu potwierdzic rzeczywistych Hz, W ani temperatury PMT.
Poprawnosc tych przeliczen sprawdzono syntetycznie, nie na fizycznym PMT.

Regresje repozytorium przeszly. Dodatkowy test wykazal niespojnosc PTL:
native ignoruje licznik bandwidth `0xc20`, ktory standalone sumuje z `0xc18`.
To problem reprodukowany syntetycznie; na tym hoscie nie ma fizycznego PTL.

## Checkout i zasady

- Zadana sciezka `/home/fst/okomor/native/edge-ai-libraries` nie istnieje.
- Uzyto workspace `/home/fst/okomor/edge-ai-libraries`.
- Branch pozostawiono bez zmian: `test/npu-metrics`.
- HEAD: `7fec5540bcae896f7ef2c41faff732555543a1fd`.
- Branch poprawek: `origin/mkobyli-fad-metrics-manager`, commit `1c0b429e`.
- `git merge-base --is-ancestor origin/mkobyli-fad-metrics-manager HEAD`: exit 0.
- Roznica wzgledem brancha poprawek: tylko sledzony raport PTL, 432 linie.
- Poczatkowy status: `?? test-result-mtl.md`; plik byl pusty i nie byl zmieniany.
- Nie zmieniono kodu produkcyjnego, brancha, sterownika, firmware ani uprawnien.
- Nie restartowano istniejacych uslug, nie zmieniano ustawien zasilania.
- Nie wykonano commit/push/prune ani operacji na istniejacych wolumenach.

Przeczytano instrukcje root, AGENTS i instrukcje ViPPET, README Metrics Manager,
aktualny kod collector/PMT/CLI, standalone monitor/exporter, API/SSE,
harness/reporters oraz selektory i testy UI przed odpowiednimi testami.

## Sprzet i dostep

- CPU: Intel Core Ultra 7 165H, family 6, model 170, stepping 4.
- 22 logiczne CPU, 16 rdzeni, x86_64; brak flagi hypervisor w `lscpu`.
- Kernel: `6.17.0-23-generic`, Ubuntu, PREEMPT_DYNAMIC.
- PCI: `0000:00:0b.0`, Intel Meteor Lake NPU, `8086:7d1d`, revision 04.
- Sterownik: `intel_vpu`, wersja `1.0.0 6.17.0-23-generic`, in-tree.
- Srcversion sterownika: `7F053039ED3AE4C72B280DE`.
- Firmware na dysku: `/lib/firmware/intel/vpu/vpu_37xx_v1.bin.zst`, 426466 B.
- SHA256 firmware: `0c3dac41445653d1a8fc95b17d14366a982d112bfd1a6bb664b9ec763abcbfd8`.
- Firmware nie zmieniano; wersji aktywnego firmware z debugfs nie sprawdzano.
- Urzadzenie: `/dev/accel/accel0`, major/minor 261:0, `root:render`, `0660`.
- Konto: `fst`, UID/GID 1000, czlonek `render` (992) i `docker`.
- `/sys/class/accel/accel0/device` wskazuje fizyczny PCI NPU.

Wynik odczytow przed obciazeniem:

| Wezel pod `/sys/class/accel/accel0/device` | Dostep |        Wartosc |
| :----------------------------------------- | :----- | -------------: |
| `npu_busy_time_us`                         | 0444   | 97919250295 us |
| `npu_memory_utilization`                   | 0444   |     68722688 B |
| `npu_current_frequency_mhz`                | 0444   |          0 MHz |
| `npu_max_frequency_mhz`                    | 0444   |       1400 MHz |

Wszystkie powyzsze wezly sa obecne i czytelne przez `fst`.
MTL rzeczywiscie udostepnia pamiec: 68722688 B = 65.5390625 MiB.
Pamiec nie moze byc warunkowana sztywno generacja PTL.

Dostepne PMT, odczytane z metadanych sysfs:

| Wezel   | GUID       | Size B | Offset B |
| :------ | :--------- | -----: | -------: |
| telem1  | 0x130670b2 |   3352 |      744 |
| telem2  | 0x1a067002 |   1216 |      200 |
| telem3  | 0x130671b2 |    596 |        0 |
| telem4  | 0x1a067102 |    132 |        0 |
| telem5  | 0x2625030  |   1180 |     3200 |
| telem6  | 0x2781795  |    196 |      288 |
| telem7  | 0x2880208  |    188 |     3584 |
| telem8  | 0x3224788  |    932 |     1024 |
| telem9  | 0x5077612  |    844 |     3344 |
| telem10 | 0x5317874  |     48 |      104 |
| telem11 | 0x5709395  |     80 |      512 |
| telem12 | 0x5850976  |    608 |      512 |
| telem13 | 0x4772882  |    292 |     1024 |

Metadane `guid`, `size`, `offset` sa czytelne. Wszystkie binarne `telem` maja
`root:root 0440` i sa nieczytelne dla `fst`. Wlasciwy layout MTL `telem1`
jest obecny, ale odczyt `telem1/telem` daje `permission denied`.
Nie uzywano sudo, privileged container ani obchodzenia uprawnien.

## Srodowiska testowe

Katalog roboczy: `/tmp/npu-mtl-verify.sfKZ4DmA`, potem usuniety.

- Go 1.26.0, linux/amd64, systemowy GCC dla `-race`.
- Go pobrano do temp i zweryfikowano z pelnym oficjalnym indeksem SHA256.
- SHA256 archiwum Go: `aac1b08a0fb0c4e0a7c1555beb7b59180b05dfc5a3d62e40e9de90cd42f88235`.
- Osobne `GOCACHE`, `GOMODCACHE`, `GOPATH`; `GOFLAGS=-mod=readonly`.
- Python 3.12.3, tymczasowy venv; bez globalnej instalacji pakietow.
- FastAPI 0.143.1, uvicorn 0.54.0, Pydantic 2.14.3, settings 2.16.2.
- pytest 9.2.0, pytest-asyncio 1.5.0, httpx 0.28.1, PyYAML 6.0.3.
- Telegraf 1.40.3, git `41b45388`, binarka skopiowana z lokalnego obrazu.
- Node 24.22.3 i npm 11.19.1, archiwum sprawdzone przez oficjalny SHASUMS256.
- UI: kopia temp, `npm ci` z obecnego package-lock, bez zmian oryginalu.

Nie uzyto starej binarki kolektora z obrazu. Collector i TUI zbudowano z HEAD.
SHA256 zbudowanego `mm-plugin-npu`:
`d2c62c519d7a1efdec428f012279b88ab323b6cc83bd952d337665bd6b987da9`.

Hostowy OpenVINO 2026.4.1 widzial tylko `CPU`; odczyt NPU property zakonczyl
sie bledem `Unsupported configuration key: FULL_DEVICE_NAME`.
Nie diagnozowano tego przez wymiane UMD/sterownika. Lokalny runtime ViPPET
`intel/vippet-app:2026.3.0-dev`, image ID `63969b29e232`, widzial `CPU`, `NPU`.
To jego OpenVINO 2026.5.0-23697-32776832f8d wykonal fizyczny test.

## Wyniki regresji

Liczby Go oznaczaja funkcje top-level; podtesty podano osobno.

| Zakres                                             | Pass | Fail | Skip | Wynik                        |
| :------------------------------------------------- | ---: | ---: | ---: | :--------------------------- |
| Native NPU, osobno                                 |   26 |    0 |    0 | 95 pass z podtestami         |
| Caly modul Go, `-race`                             |  452 |    0 |    1 | 709 pass z podtestami        |
| Standalone monitor/exporter                        |   41 |    0 |    0 | offline pytest               |
| API/SSE/capabilities/selection/models              |  334 |    0 |    0 | zawiera rzeczywisty Telegraf |
| Harness i reportowanie                             |  167 |    0 |    0 | cztery pliki unit            |
| UI, testy repozytorium                             |    5 |    0 |    0 | selektory, history, SSR      |
| API capture do UI, dodatkowe                       |    8 |    0 |    0 | 4 stany x 2 generacje        |
| Live collector/Telegraf/API/harness/TUI, dodatkowe |    2 |    0 |    0 | syntetyczne MTL/PTL          |
| Drugi licznik bandwidth PTL, dodatkowy             |    0 |    1 |    0 | reprodukcja problemu ponizej |

`go vet ./...`: exit 0, bez diagnostyki. Collector/TUI build: exit 0.
UI `npm run build`: exit 0; ostrzezenia o duzym bundle i mieszanym imporcie
dynamic/static, bez bledu zwiazanego z NPU.

Jedyny skip Go: `TestSignalChild`, powod `subprocess helper`; to pomocniczy
test uruchamiany przez test nadrzedny, nie brak sprzetu.
12 pakietow Go przeszlo; 6 pakietow bez plikow testowych nie oznacza 6 skipow testow.
Osobne 26 testow NPU jest juz zawarte w 452 testach calego modulu: nie sumowac dwa razy.

Pierwszy przebieg API mial 330 pass/4 skip z powodu braku
`TELEGRAF_TEST_BINARY`. Po ustawieniu zmiennej ten sam zakres mial 334 pass/0 skip.
Trzy testy wygaszania NPU i cztery testy wyboru pluginow uzywaly rzeczywistego Telegrafa.

Pierwszy dodatkowy test live mial 3 fail: dwa dotyczyly synchronizacji/sprzatania
skryptu testowego, jeden licznika PTL. Po poprawie tylko skryptu temp:
2 pass/1 fail. Bramka aktywnosci czeka na pelny interwal po recovery;
Telegraf dostal do 12 s na zamkniecie. To nie byly poprawki produkcyjne.

## Fizyczny idle/load/recovery

Kontener testowy: nieuprzywilejowany UID/GID 1000, dodatkowa grupa 992,
`--cap-drop ALL`, `no-new-privileges`, `--network none` i tylko urzadzenie NPU.
Biezacy kolektor oraz skrypt byly zamontowane read-only. Obraz runtime nie byl przebudowywany.

Model: FP32 MatMul `[1,512,512]` z diagonalna waga 0.5, nastepnie ReLU.
Wybor: `core.compile_model(model, "NPU")`, bez AUTO/HETERO/fallback.
Potwierdzenie: `EXECUTION_DEVICES = ["NPU.0"]` i numeryczny wynik `allclose`: PASS.
NPU: `Intel(R) AI Boost`, architektura `3720`, driver property `1789049546`.

Fazy: idle 5 s, load 10 s, recovery 5 s. Przygotowanie modelu mialo limit 30 s
i bylo osobna faza; zebrano w nim jedna dodatkowa probke.
W load wykonano **16428 inferencji w 10.0000630915 s**.
Nie wykonywano dalszego stress testu, resetow ani wymuszania rollover na sprzecie.

| Faza     | Probki | Czestotliwosc sterownika MHz | Utilization %          | Pamiec MiB |
| :------- | -----: | :--------------------------- | :--------------------- | :--------- |
| Idle     |     10 | 0-633, potem 0               | 0; pierwsza brak delty | 65.5390625 |
| Prepare  |      1 | 0                            | 0                      | 65.5390625 |
| Load     |     20 | 1333-1400, avg 1396.65       | 54-66, avg 65.05       | 79.734375  |
| Recovery |     10 | 0                            | pierwsza 11, potem 0   | 65.5390625 |

Pierwsza idle probka 633 MHz byla przejsciowa po otwarciu runtime;
nie ma podstaw, by przypisywac ja inferencji. Pierwsza recovery delta obejmuje
koncowke load, dlatego 11% nie oznacza nieudanego recovery.

Rzeczywiste odstepy startow odczytu: 0.500295459-0.501272410 s.
Sam odczyt czterech sysfs trwal 0.000177426-0.001152317 s.
Utilization liczono `min(100, int(100 * delta_busy_us / elapsed_s / 1e6))`.
Pierwsza probka nie miala baseline: utilization bylo null, nie sztuczne zero.

Reprezentatywne **surowe sysfs**, nie metryki PMT kolektora:

| Faza     |      Monotonic ns |   Elapsed s |     Busy us | Memory B | Frequency MHz | Util % |
| :------- | ----------------: | ----------: | ----------: | -------: | ------------: | -----: |
| Idle     | 11650406184902205 | 0.500639819 | 97919250295 | 68722688 |             0 |      0 |
| Load     | 11650414193471823 | 0.500297216 | 97922822970 | 83607552 |          1400 |     66 |
| Recovery | 11650421700883742 | 0.500551470 | 97925860865 | 68722688 |             0 |      0 |

Pamiec wzrosla o 14884864 B = 14.1953125 MiB i wrocila do baseline.
Nie wymuszano pamieci rownej zero: baseline 65.539 MiB pozostalo prawdziwym odczytem.

Kolektor `-once`: exit 1, stdout pusty:

```text
mm-plugin-npu: reading telemetry from /sys/class/intel_pmt/telem1/telem: open /sys/class/intel_pmt/telem1/telem: permission denied
```

Kolektor ciagly podczas calego testu: **0 linii**, jedno ostrzezenie o idling,
po SIGTERM exit 0. Wszystkie jego pomiary frequency/power/temperature/bandwidth/
memory/utilization pozostaly niedostepne. Braku nie interpretowano jako zero.
Brak PMT blokuje konstruktor przed eksportem nawet czytelnych sysfs; to obecny,
udokumentowany model uprawnien, a nie brak pamieci MTL.

## Kontrakt jednostek i konsumenci

| Pole API                | Kontrakt     | Syntetyczna kontrola                       | Fizyczna kontrola                         |
| :---------------------- | :----------- | :----------------------------------------- | :---------------------------------------- |
| `npu_frequency`         | Hz           | MTL ratio 30 -> 1e9; PTL ratio 40 -> 1e9   | PMT zablokowane                           |
| `npu_power`             | W            | delta joules / real elapsed, ok. 4 W       | zablokowane                               |
| `npu_temperature`       | C            | 45                                         | zablokowane                               |
| `npu_bandwidth`         | decimal MB/s | PTL pierwszy licznik ok. 10; MTL brak pola | MTL layout bez licznika                   |
| `npu_memory_mb`         | binary MiB   | 1048577 B -> 1.0000009536743164            | sysfs potwierdza MiB, eksport zablokowany |
| `npu_frequency_max_mhz` | MHz          | 1800 bez dalszego skalowania               | driver 1400, eksport zablokowany          |
| `npu_utilization`       | %            | ok. 50, reset/brak/recovery                | delta sysfs 54-66 podczas load            |

Wzory w aktualnym kodzie i testach:

- MTL/ARL: `PLL ratio * 50e6 * 2/3`; ratio 48 daje 1600000000 Hz.
- PTL: `PLL ratio * 50e6 / 2`; ratio 78 daje 1950000000 Hz.
- MTL `memoryBW=-1`: brak eksportu bandwidth, nie zero.
- Pamiec: obecnosc sysfs na MTL/PTL, dzielenie tylko przez `1024*1024`.
- Bandwidth: delta licznika decimal kB / real seconds / 1000.
- Brak aliasow `npu_frequency_hz` i `npu_power_w` w przechwyconych API/UI payloadach.

Rollover bandwidth 32-bit, power/busy reset, bledne/nonpositive interwaly,
truncated PMT oraz missing/invalid sysfs sprawdzono testami offline.
Nie wymuszano tych zdarzen na fizycznym urzadzeniu.

Dodatkowe live testy mialy syntetyczny sysfs, ale **rzeczywiste procesy**
biezacego kolektora, Telegrafa, uvicorn/API, harness i `mm-tui -once`.
Wykorzystano wolne porty 127.0.0.1 i wygaszanie 1 s w izolowanym Telegrafie;
nie zmieniano produkcyjnej konfiguracji wygaszania 15 s.

Przechwycone reprezentatywne API:

```json
{"platform":"synthetic MTL","npu_frequency":1000000000,"npu_frequency_max_mhz":1800,"npu_memory_mb":1.0000009536743164,"npu_power":3.926691715090861,"npu_temperature":45,"npu_utilization":49}
{"platform":"synthetic PTL","npu_bandwidth":9.831569968799114,"npu_frequency":1000000000,"npu_frequency_max_mhz":1800,"npu_memory_mb":1.0000009536743164,"npu_power":3.9324606549926115,"npu_temperature":45,"npu_utilization":49}
```

Harness Hz->MHz: 1000000000 -> 1000.0, pojedyncza konwersja.
Pamiec i bandwidth zachowuja wartosci; zaokraglenie agregatu 1.0 MiB nie jest druga konwersja.
MTL nie mial klucza bandwidth. PTL TUI pokazal `9.9 MB/s`, `1 MiB`, `1.00 GHz of 1.80 GHz`.

Kazdy stan mial cztery przechwycone probki, takze gdy kolejne SSE zawieraly ten sam timestamp.
`sample_count` to liczba odpytywan harnessu, nie liczba unikalnych timestampow kolektora.
Przy obecnym NPU: `sample_count=4`, `npu_settle_samples_dropped=1`, `npu_sample_count=3`.
Po zerach: status `idle`, zera zachowane w API/harness/TUI i selektorach UI.
Po zaniku PMT: NPU wygaslo z API, status `no_samples`, `npu_sample_count=0`,
`npu_settle_samples_dropped=0`; CPU pozostalo. TUI nie tworzylo zerowego panelu NPU.
Recovery przywrocilo status `ok`, 1000 MHz i poprawne jednostki.

UI przetestowano przez testy SSR/history/indicators i osiem odpowiedzi live API
podanych do jego reducer/selektorow. UI nie konsumuje pamieci ani bandwidth NPU;
nie ma wiec wizualnego potwierdzenia tych dwoch pol w UI. TUI/harness je obsluguja.

## Wykryty problem PTL

**Niespojnosc agregacji drugiego licznika bandwidth, reprodukcja syntetyczna.**

Native
[pmt.go](microservices/metrics-manager/native/internal/npu/pmt.go#L69)
ma dla PTL tylko `memoryBW: 0xc18`;
[BandwidthBytes](microservices/metrics-manager/native/internal/npu/pmt.go#L208)
czyta jeden word. Standalone
[get_ptl_regs](tools/npu-monitor-tool/npu-monitor-tool.py#L83)
deklaruje `[0xc18, 0xc20]`, a
[get_noc_bandwidth](tools/npu-monitor-tool/npu-monitor-tool.py#L275)
sumuje oba.

Reprodukcja: synthetic GUID `0x3086000`, PMT 4096 B, pierwszy licznik stale 0,
drugi `0 -> 20000` podczas interwalu ok. 0.5 s. Native oddaje:

```text
npu,host=gklab-157-116 power=0,frequency=1000000000,temperature=0i,bandwidth=0,tile_config=0i,utilization=0i,memory_mb=1.0000009536743164,power_state=0i,frequency_max_mhz=1800 1794559782868586477
```

Standalone widzi wzrost 20.0 decimal MB, czyli ok. 40 MB/s dla 0.5 s.
Dodatkowa asercja, ze native nie odrzuca tej aktywnosci, FAIL.
Testy repozytorium nie pokrywaja drugiego licznika PTL.

Proponowana poprawka, niezaimplementowana: potwierdzic oba offsety w docelowym
layout PMT PTL; reprezentowac per-generation liste licznikow, liczyc delty
i rollover osobno dla kazdego, potem sumowac decimal MB/s. Podniesc minimalny
rozmiar snapshotu dla ostatniego offsetu. Dodac test ruchu tylko w `0xc20`,
jednoczesnego ruchu i niezaleznego rollover; pozostawic MTL bez bandwidth.
Nie uznawac tej reprodukcji za fizyczne potwierdzenie layoutu PTL.

Minimalna reprodukcja po zbudowaniu aktualnego `mm-plugin-npu`:

```python
import pathlib
import struct
import subprocess
import tempfile

binary = "/path/to/fresh/mm-plugin-npu"
with tempfile.TemporaryDirectory(prefix="ptl-counter-repro-") as temporary:
    root = pathlib.Path(temporary)
    pmt = root / "class/intel_pmt/telem1"
    device = root / "bus/pci/drivers/intel_vpu/0000:00:0b.0"
    pmt.mkdir(parents=True)
    device.mkdir(parents=True)
    for name, value in {"guid": "0x3086000", "size": "4096", "offset": "0"}.items():
        (pmt / name).write_text(value)
    buffer = bytearray(4096)
    struct.pack_into("<Q", buffer, 0x18, 40)
    (pmt / "telem").write_bytes(buffer)
    process = subprocess.Popen(
        [binary, "-once", "-interval", "500ms", "-sysfs-root", temporary],
        stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True,
    )
    try:
        print(process.stderr.readline().strip())
        struct.pack_into("<Q", buffer, 0xc20, 20000)
        replacement = pmt / "snapshot.new"
        replacement.write_bytes(buffer)
        replacement.replace(pmt / "telem")
        stdout, stderr = process.communicate(timeout=3)
        print(stdout, stderr)
        assert "bandwidth=0," not in stdout, "Second PTL counter discarded"
    finally:
        if process.poll() is None:
            process.terminate()
            process.wait(timeout=3)
```

## Komendy wykonane

Ponizej najwazniejsze komendy; `WORK` oznacza usuniety katalog temp.
Skrypty `physical_check.py`, `test_live_path.py` i UI capture byly tylko w temp.
Przy ponownym tescie trzeba odtworzyc fixture/skrypt, nie korzystac ze starych binarek.

```bash
REPO=/home/fst/okomor/edge-ai-libraries
WORK=/tmp/npu-mtl-verify.sfKZ4DmA
git -C "$REPO" branch --show-current
git -C "$REPO" log -1 --format='%H %cI %s'
git -C "$REPO" status --short
git -C "$REPO" merge-base --is-ancestor origin/mkobyli-fad-metrics-manager HEAD
git -C "$REPO" diff --stat origin/mkobyli-fad-metrics-manager..HEAD
lscpu
uname -a
lspci -nnk
modinfo intel_vpu
ls -l /dev/accel /sys/class/accel /sys/class/intel_pmt
cat /sys/class/accel/accel0/device/npu_busy_time_us
cat /sys/class/accel/accel0/device/npu_memory_utilization
cat /sys/class/accel/accel0/device/npu_current_frequency_mhz
cat /sys/class/accel/accel0/device/npu_max_frequency_mhz
docker ps --format '{{.ID}} {{.Names}} {{.Image}} {{.Ports}}'
systemctl is-active telegraf metrics-manager metrics-manager-npu

export PATH="$WORK/bin:$WORK/go/bin:$PATH"
export GOCACHE="$WORK/gocache" GOMODCACHE="$WORK/gomodcache"
export GOPATH="$WORK/gopath" GOTOOLCHAIN=local GOFLAGS=-mod=readonly
cd "$REPO/microservices/metrics-manager/native"
go test -json -count=1 ./internal/npu
go test -race -json -count=1 ./...
go vet ./...
go build -o "$WORK/bin/mm-plugin-npu" ./cmd/mm-plugin-npu
go build -o "$WORK/bin/mm-tui" ./cmd/mm-tui
"$WORK/bin/mm-plugin-npu" -once

cd "$REPO/tools/npu-monitor-tool"
"$WORK/venv/bin/python" -m pytest -q -ra -p no:cacheprovider tests
cd "$REPO/microservices/metrics-manager"
export TELEGRAF_TEST_BINARY="$WORK/bin/telegraf" PYTHONDONTWRITEBYTECODE=1
"$WORK/venv/bin/python" -m pytest -q -ra -p no:cacheprovider \
  tests/test_telegraf_integration.py tests/test_sse.py tests/test_routes.py \
  tests/test_telemetry_selection.py tests/test_capabilities_registry.py \
  tests/test_models.py
cd "$REPO/tools/visual-pipeline-and-platform-evaluation-tool/vippet"
PYTHONPATH="$PWD" "$WORK/venv/bin/python" -m pytest -q -ra -p no:cacheprovider \
  tests/unit/hw_monitor_test.py tests/unit/perf_helpers_test.py \
  tests/unit/performance_reporters_test.py tests/unit/reporters_test.py

docker run --rm --name npu-mtl-physical-test \
  --user 1000:1000 --group-add 992 --cap-drop ALL \
  --security-opt no-new-privileges --network none \
  --device /dev/accel/accel0:/dev/accel/accel0:rw \
  --mount "type=bind,src=$WORK,dst=/work,readonly" \
  --env PYTHONDONTWRITEBYTECODE=1 --env HOME=/tmp --env XDG_CACHE_HOME=/tmp \
  --entrypoint python3 intel/vippet-app:2026.3.0-dev /work/physical_check.py

cd "$WORK"
./venv/bin/python -m pytest -q -ra -p no:cacheprovider \
  --basetemp="$WORK/pytest-live" test_live_path.py
cd "$WORK/ui"
npm ci --cache "$WORK/npm-cache" --no-audit --no-fund
npm test
npm run build
node --experimental-strip-types --test tests/api-capture.test.mjs
```

## Niewykonane kontrole i ograniczenia

- Fizyczny PLL ratio i porownanie kolektora do driver frequency: brak odczytu PMT.
- Fizyczna power/temperature i ich zmiany idle/load/recovery: ta sama blokada.
- Fizyczny eksport pamieci/max/utilization: konstruktor kolektora wymaga PMT.
- Fizyczny bandwidth PTL i oba jego liczniki: host MTL, brak sprzetu PTL.
- Pelny fizyczny collector -> Telegraf -> API -> UI/harness: brak wejscia PMT.
- Pelny backend ViPPET i browser E2E: nie uruchamiano calego stosu DLStreamer/model services.
- UI wizualne pamiec/bandwidth NPU: brak tych pol w obecnych selektorach UI.
- GPU/CPU inference smoke i package-install test: poza zakresem, brak fallbacku i brak instalacji uslug.
- Globalny stress, device reset, wymuszanie wrap, zmiany power policy: celowo zabronione.

Nie ma podstaw do oznaczenia tego hosta jako fizycznie zweryfikowanego PTL
ani do wydania pelnego PASS dla fizycznych metryk NPU kolektora.

## Sprzatanie

Wlasne kontenery uruchamiano z `--rm`; kontener do kopiowania narzedzia usunieto.
Procesy testowego Telegrafa/API/kolektora zakonczono. Obrazy istniejace przed
testami zachowano; nic nie instalowano globalnie.
Usunieto temp venv, Go/Node, buildy, npm/cache, skrypty, kopie UI i logi robocze.
Zachowano tylko ten raport; oryginalny pusty raport MTL pozostal nietkniety.

Odstepstwo od pelnej izolacji artefaktow: pierwsze suite'y pytest korzystaly
z domyslnego `/tmp/pytest-of-fst`, a nie jawnego `--basetemp`.
Pozostale po rotacji katalogi wlasnych przebiegow to `pytest-47`, `pytest-48`, `pytest-49`.
Nie zinwentaryzowano wczesniejszej zawartosci tego katalogu, wiec skutkow
standardowej automatycznej rotacji pytest nie mozna niezaleznie potwierdzic.
Reczne sprzatanie ograniczono do tych zidentyfikowanych wlasnych katalogow
i symlinka wskazujacego na wlasny `pytest-49`; katalog nadrzedny pozostawiono.
Pozniejsze live testy uzywaly jawnego `--basetemp` wewnatrz `WORK`.

Pip korzystal z domyslnego wspoldzielonego cache pobran uzytkownika.
Nie zinwentaryzowano go przed instalacja, wiec pozostawiono ten cache zamiast
ryzykowac usuniecie cudzych wpisow. Pakiety zainstalowano tylko w usunietym venv.
Przy powtorzeniu testow nalezy ustawic `PIP_CACHE_DIR` i `--basetemp` wewnatrz
nowego katalogu temp od pierwszej komendy.
