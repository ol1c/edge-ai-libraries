<!-- SPDX-FileCopyrightText: (C) 2026 Intel Corporation -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Raport testow NPU: Metrics Manager i ViPPET na Panther Lake

Data testow: 2026-11-13.

## Podsumowanie

Host zostal zidentyfikowany jako **Panther Lake (PTL), nie Meteor Lake (MTL)**.
Identyfikacja opiera sie na CPU, PCI ID NPU i GUID PMT, nie na nazwie hosta.

Regresje zakonczyly sie bez bledow po uzupelnieniu zaleznosci testowych.
Potwierdzono fizyczne wykonanie OpenVINO wylacznie na NPU oraz reakcje
czytelnych licznikow sysfs na krotkie obciazenie i recovery.

Pelna fizyczna walidacja PMT pozostaje **zablokowana przez uprawnienia**.
Aktualny kolektor zostal zbudowany z checkoutu, ale nie mogl odczytac PMT.
Wykryto jedna syntetycznie odtworzona niespojnosc bandwidth PTL pomiedzy
kolektorem Go a narzedziem standalone. Nie zmieniano kodu produkcyjnego.

## Repozytorium i zasady wykonania

- Uzyte repozytorium: `/home/user/okomor/edge-ai-libraries`.
- Podana sciezka `/home/fst/okomor/native/edge-ai-libraries` nie byla dostepna;
	zgodnie z poleceniem uzyto biezacego workspace.
- Branch: `mkobyli-fad-metrics-manager`.
- HEAD: `1c0b429e21f4e9085a3d484475131c1bc9bfaf70`.
- Status przed testami i po sprzataniu: czysty.
- Przed zapisaniem tego raportu jedynym nie sledzonym plikiem byl ten raport.
- Przeczytano instrukcje repozytorium, instrukcje ViPPET i aktualny kod
	kolektora, standalone tools, API, harnessu/reportowania i UI.
- Nie przelaczano brancha, nie nadpisywano lokalnych zmian, nie commitowano
	ani nie pushowano.
- Nie restartowano istniejacych uslug, nie zmieniano sterownika, firmware,
	uprawnien urzadzen ani ustawien zasilania.
- Nie wymuszano resetu urzadzenia ani zawijania licznikow na sprzecie.
- Narzedzia, zaleznosci, fixture, kopia UI i cache znajdowaly sie w izolowanym
	katalogu tymczasowym. Nie uzywano starego obrazu Docker ani Docker prune.

## Konfiguracja sprzetowa

| Element | Odczyt |
| --- | --- |
| CPU | Intel Core Ultra X7 358H |
| CPUID | Family 6, model 204, stepping 2 |
| CPU logiczne | 16 |
| Kernel | `7.0.0-31-generic` |
| Sterownik NPU | `intel_vpu`, wersja `1.0.0 7.0.0-31-generic` |
| PCI NPU | `0000:00:0b.0`, `8086:b03e`, revision `04` |
| Urzadzenie | `/dev/accel/accel0`, major/minor `261:0` |
| Uprawnienia urzadzenia | `660 root:render`; uzytkownik nalezy do `render` |
| PMT identyfikujace PTL | `telem2`, GUID `0x3086000`, 3352 bajty |

CPU model 204, PCI ID `8086:b03e` i GUID `0x3086000` zgodnie identyfikuja PTL.

### Dostepne PMT

| Wezel | GUID | Rozmiar [B] | Binarny `telem` czytelny |
| --- | --- | ---: | --- |
| `telem1` | `0x3086100` | 744 | Nie |
| `telem2` | `0x3086000` | 3352 | Nie |
| `telem3` | `0x47179370` | 644 | Nie |
| `telem4` | `0x33405810` | 240 | Nie |
| `telem5` | `0x25734732` | 1280 | Nie |
| `telem6` | `0x33302935` | 720 | Nie |
| `telem7` | `0x4772882` | 292 | Nie |
| `telem8` | `0x12437394` | 80 | Nie |
| `telem9` | `0x17491217` | 320 | Nie |
| `telem10` | `0x32321517` | 156 | Nie |

Atrybuty `guid`, `size` i `offset` byly czytelne. Binarny wezel PMT PTL
mial `440 root:root`, a proba odczytu zwrocila `Permission denied`.
Nie podnoszono uprawnien ani nie obchodzono tej blokady.

### Dostepnosc sysfs NPU

Sciezka bazowa: `/sys/class/accel/accel0/device`.

| Atrybut | Dostepnosc | Wartosci idle |
| --- | --- | --- |
| `npu_busy_time_us` | Czytelny, `444 root:root` | `0` przed obciazeniem |
| `npu_memory_utilization` | Czytelny, `444 root:root` | `68714496` B |
| `npu_current_frequency_mhz` | Czytelny | `0` MHz |
| `npu_max_frequency_mhz` | Czytelny, `444 root:root` | `2050` MHz |
| `npu_min_frequency_mhz` | Brak wezla | Niedostepne, nie zero |
| `power_state` | Czytelny | `D3hot` |

Pamiec idle: `68714496 / 1024 / 1024 = 65.53125 MiB`.

## Narzedzia i izolacja

Na poczatku brakowalo Go, pip, pytest, OpenVINO, Telegrafa, Node/npm i Dockera.
Nie wykonywano instalacji globalnej ani zmian sterownika/userspace NPU.

**Docker nie byl uzywany.** Polecenie `docker ps` zwrocilo `command not found`
(kod wyjscia 127). Nie budowano ani nie uruchamiano obrazow Docker,
nie tworzono kontenerow ani wolumenow. Kolektor zbudowano bezposrednio
z checkoutu, a testy wykonano w tymczasowym venv i z przenosnymi narzedziami
Go, Telegraf oraz Node.

Do `/tmp/npu-validation-5QMDcMch` pobrano lub zainstalowano:

- Python venv na Python 3.12.3 i przenosny pip 26.3.
- Zaleznosci API/testow zadeklarowane w Metrics Manager.
- Zaleznosci standalone exportera, w tym Flask.
- Go 1.27.4, z SHA-256 sprawdzonym wedlug oficjalnego manifestu wydania.
- Telegraf 1.40.3, z SHA-256 przypietym w repozytorium:
	`cbb95c43303db6e75e129a4eb8888ab1468009e61f6615e64a1cb82f50b0f4ce`.
- OpenVINO `2026.4.1-22982-e213a147257-releases/2026/4`.
- Node LTS 24.22.3, z SHA-256 sprawdzonym wedlug oficjalnego manifestu.
- Tymczasowa kopia UI; `npm ci --ignore-scripts --no-audit --no-fund`.

Cache Go, pip, npm, raporty testow i fixture byly lokalne dla tego katalogu.
Testy Python uruchamiano z `PYTHONDONTWRITEBYTECODE=1`, bez cache pytest,
z `--basetemp` i raportami JUnit w katalogu tymczasowym.

## Wyniki regresji

Liczby Go obejmuja przypadki koncowe, bez ponownego liczenia rodzicow subtestow.
Wyniki ponizej dotycza koncowych, poprawnie skonfigurowanych uruchomien.

| Zakres | Pass | Fail | Skip |
| --- | ---: | ---: | ---: |
| Natywny NPU i CLI | 83 | 0 | 0 |
| Pelne Go z `-race`, w tym TUI | 662 | 0 | 1 |
| Standalone monitor/exporter | 41 | 0 | 0 |
| Harness i reportowanie | 326 | 0 | 0 |
| Metrics Manager/API | 661 | 0 | 56 |
| UI NPU | 2 | 0 | 0 |
| Dodatkowa syntetyczna sciezka MTL/PTL | 2 | 0 | 0 |

Harness dodatkowo: **99 poprawnych subtestow**.
`go vet ./...` zakonczyl sie bez diagnostyki.

Go raportowalo 26 poprawnych testow najwyzszego poziomu dla NPU/CLI oraz
452 poprawne testy najwyzszego poziomu i 1 skip dla pelnego `-race`.
Liczby te nie sa dodatkowymi przypadkami do sumowania z tabela.

### Pominiecia i poczatkowe blokady zaleznosci

- Skip Go: `cmd/mm-tui/TestSignalChild`, powod `subprocess helper`.
- 56 skip API: `helm CLI not installed`; dotycza testow Helm, nie NPU.
- Pierwszy standalone run: 31 pass i 10 setup errors z powodu braku Flask.
	Po instalacji deklarowanych zaleznosci exportera: 41 pass, bez bledow.
- Pierwszy API run: 657 pass, 60 skip. Cztery skip wymagaly ustawienia
	`TELEGRAF_TEST_BINARY`; po jego ustawieniu pozostalo tylko 56 skip Helm.
- Nie traktowano poczatkowych brakow zaleznosci jako regresji produkcyjnej.

### Rzeczywisty Telegraf

Testy API korzystaly z prawdziwego Telegrafa 1.40.3 na tymczasowych portach.
Suite integracji Telegrafa: 23 poprawne testy.

Test wygasania niedostepnych pol NPU, zachowania rzeczywistych zer i recovery
przeszedl dla wszystkich trzech konfiguracji eksportera:

- Metrics Manager.
- Bare-metal.
- ViPPET.

Dane w tych testach byly syntetyczne; nie byly fizycznymi probkami PMT.

## Wykonane komendy

Ponizsze zmienne opisuja wykorzystane sciezki. Katalog `TMP` zostal po testach
usuniety; lista dokumentuje wykonanie, nie stanowi gotowego skryptu odtworzenia
usunietych narzedzi i dodatkowych fixture.

```bash
REPO=/home/user/okomor/edge-ai-libraries
TMP=/tmp/npu-validation-5QMDcMch
MM="$REPO/microservices/metrics-manager"
VP="$REPO/tools/visual-pipeline-and-platform-evaluation-tool/vippet"

git -C "$REPO" branch --show-current
git -C "$REPO" rev-parse HEAD
git -C "$REPO" status --short
uname -a
lscpu
lspci -nnk
id
modinfo intel_vpu

cd "$MM/native"
export PATH="$TMP/go/bin:$PATH"
export GOCACHE="$TMP/go-cache" GOMODCACHE="$TMP/go-mod"
export GOPATH="$TMP/go-path" GOTMPDIR="$TMP" GOTOOLCHAIN=local
go test -count=1 -json ./internal/npu ./cmd/mm-plugin-npu
go test -race -count=1 -json ./...
go vet ./...
go build -o "$TMP/mm-plugin-npu" ./cmd/mm-plugin-npu
timeout 10s "$TMP/mm-plugin-npu" -once -interval 500ms

export PYTHONDONTWRITEBYTECODE=1 TMPDIR="$TMP"
cd "$REPO"
"$TMP/venv/bin/python" -m pytest -q -p no:cacheprovider \
	--basetemp="$TMP/pytest-standalone-final" \
	--junitxml="$TMP/standalone-final.xml" \
	tools/npu-monitor-tool/tests/test_npu_monitor.py

cd "$VP"
PYTHONPATH="$VP" "$TMP/venv/bin/python" -m pytest -q -p no:cacheprovider \
	--basetemp="$TMP/pytest-harness" --junitxml="$TMP/harness.xml" \
	tests/unit/hw_monitor_test.py tests/unit/perf_helpers_test.py \
	tests/unit/performance_*test.py tests/unit/reporters_test.py

cd "$MM"
export PATH="$TMP/telegraf-1.40.3/usr/bin:$PATH"
export TELEGRAF_TEST_BINARY="$TMP/telegraf-1.40.3/usr/bin/telegraf"
"$TMP/venv/bin/python" -m pytest -q -rs -p no:cacheprovider \
	--basetemp="$TMP/pytest-api-final" --junitxml="$TMP/api-final.xml"

cd "$TMP/ui"
export PATH="$TMP/node-v24.22.3-linux-x64/bin:$PATH"
export npm_config_cache="$TMP/npm-cache"
npm ci --ignore-scripts --no-audit --no-fund
node --experimental-strip-types --test tests/npu-render.test.mjs

python3 "$TMP/validate_host.py"
timeout 10s python3 "$TMP/validate_host.py" "$TMP/mm-plugin-npu"
timeout --kill-after=2s 45s "$TMP/venv/bin/python" "$TMP/load_npu.py"
timeout --kill-after=2s 40s "$TMP/venv/bin/python" "$TMP/test_consumer_chain.py"
```

Pierwszy fizyczny odczyt sysfs wykonano przed utworzeniem venv. Dodatkowe
skrypty byly tymczasowymi probami testowymi, nie plikami produkcyjnymi.

## Fizyczne odczyty

### Aktualny kolektor

Kolektor zbudowano z biezacego checkoutu, bez Docker.
Diagnostyczne `-once -interval 500ms` zwrocilo kod wyjscia 1:

```text
mm-plugin-npu: reading telemetry from /sys/class/intel_pmt/telem2/telem:
open /sys/class/intel_pmt/telem2/telem: permission denied
```

Nie wyemitowal fizycznej probki. Nie zastepowano bledu sztucznymi zerami.

### Poczatkowa seria idle

Zebrano 8 probek z czytelnych sysfs, przez okolo 3.505 s.
Rzeczywiste interwaly wynosily okolo 0.5006-0.5007 s.

- Busy-time: 0 us; utilization w dostepnych interwalach: 0%.
- Czestotliwosc sterownika: 0 MHz; maksimum: 2050 MHz.
- Pamiec: 68714496 B = 65.53125 MiB.
- Power, temperature i bandwidth PMT: niedostepne, reprezentowane jako null.

### Idle/load/recovery OpenVINO

OpenVINO wykrylo `CPU`, `GPU` i `NPU`. Model skompilowano jawnie:

```python
compiled = core.compile_model(model, "NPU")
compiled.get_property("EXECUTION_DEVICES")  # "NPU"
```

Nie uzywano AUTO, HETERO ani CPU fallback.
Model testowy zawieral cztery warstwy convolution/ReLU, wejscie FP16
o ksztalcie `[1, 16, 128, 128]`, przygotowane przez OpenVINO/NumPy.

Pierwsza proba skryptu zatrzymala sie przed inferencja: OpenVINO zwrocilo
`EXECUTION_DEVICES` jako string, a skrypt oczekiwal listy. Skorygowano jedynie
tymczasowy skrypt i ponowiono test; nie byl to blad kodu produkcyjnego.

Poprawne uruchomienie: **4 s idle / 8 s load / 4 s recovery**, odpowiednio
**8 / 16 / 8 probek**. Wykonano **46704 inferencje na NPU**, bez bledow.
Po load zwolniono model i request przed pomiarem recovery.

| Reprezentatywna probka | Rzeczywisty delta-t [s] | MHz sterownika | Utilization [%] | Pamiec [MiB] |
| --- | ---: | ---: | ---: | ---: |
| Ostatnia idle | 0.500821961 | 0 | 0 | 65.53125 |
| Ostatnia load | 0.500571793 | 2050 | 49.535352 | 74.6484375 |
| Ostatnia recovery | 0.500753434 | 0 | 0 | 65.53125 |

Obserwowane zachowanie:

- Load: czestotliwosc 950-2050 MHz; po pierwszym interwale utilization
	44.28-53.24%; pamiec wzrosla do 78274560 B = 74.6484375 MiB.
- Pierwsza probka load miala utilization 0%; jej interwal obejmowal czas
	przed rozpoczeciem inferencji. Nie usuwano rzeczywistego zera.
- Pierwszy interwal recovery obejmowal koncowke load i mial okolo 47.43%.
	Nastepne siedem interwalow mialo utilization 0%.
- Recovery: pamiec wrocila do 65.53125 MiB, czestotliwosc do 0 MHz.
- Power, temperature i bandwidth pozostaly niedostepne przez cala serie;
	nie zamieniano braku odczytu w zero.

Utilization w tabeli wyliczono z delta busy-time i rzeczywistego czasu
miedzy odczytami sysfs. Nie jest to fizyczny eksport kolektora Go.

## Kontrakt jednostek i konsumenci

| Pole publiczne | Kontrakt | Sposob weryfikacji |
| --- | --- | --- |
| `npu_frequency` | Hz | Regresje, fixture MTL/PTL przez kolektor i API; fizyczny PLL zablokowany |
| `npu_power` | W | Syntetyczne delta energii / rzeczywisty interwal |
| `npu_temperature` | Stopnie Celsjusza | Syntetyczne PMT i API |
| `npu_bandwidth` | Decimal MB/s | Syntetyczne delta, interwaly i wrap; wykryta niespojnosc drugiego licznika PTL |
| `npu_memory_mb` | Binary MiB | Regresje, fixture/API oraz fizyczne bajty sysfs |
| `npu_frequency_max_mhz` | MHz | Regresje/API; fizycznie 2050 MHz |
| `npu_utilization` | Procent | Regresje oraz fizyczne delta busy-time / elapsed time |

Syntetycznie potwierdzono:

- MTL: `PLL ratio * 50e6 * 2/3`; ratio 48 daje 1600000000 Hz.
- PTL: `PLL ratio * 50e6 / 2`; ratio 82 daje 2050000000 Hz.
- MTL nie eksportuje bandwidth dla obslugiwanego layoutu PMT.
- Pamiec zalezy od dostepnosci sysfs, nie od sztywnego ograniczenia do PTL.
- Brak aliasow `npu_frequency_hz` i `npu_power_w` w zweryfikowanym API.
- Wrap, reset energii/busy-time, niepoprawne interwaly, truncated PMT,
	brak/blad odczytu, recovery i zachowanie rzeczywistych zer pokryto
	istniejacymi testami syntetycznymi, bez wymuszania ich na sprzecie.

### Izolowana pelna sciezka konsumentow

Dwa dodatkowe testy uruchomily aktualny kolektor z syntetycznym sysfs,
rzeczywisty Telegraf `inputs.execd`, rzeczywiste API Metrics Manager/SSE
i konsumenta `HardwareMonitor`. Uslugi sluchaly tylko na tymczasowych
portach `127.0.0.1` i zostaly zakonczone po testach.

Oba testy przeszly, z nastepujacymi wartosciami:

| Weryfikacja | MTL fixture | PTL fixture |
| --- | ---: | ---: |
| API `npu_frequency` [Hz] | 1600000000 | 2050000000 |
| Harness frequency [MHz] | 1600 | 2050 |
| API i surowy harness memory [MiB] | 1.0000009536743164 | 1.0000009536743164 |
| API temperature [C] | 45 | 45 |
| API power [W] | 0 | 0 |
| API utilization [%] | 0 | 0 |
| Bandwidth | Pole nieobecne | Zmierzona syntetyczna wartosc 0 |
| `sample_count` | 3 | 3 |
| `npu_sample_count` | 2 | 2 |
| `npu_settle_samples_dropped` | 1 | 1 |
| `kpi_status.NPU` | `ok` | `ok` |

Status `ok` jest zgodny z obecnym kontraktem: mimo zerowego utilization
fixture raportowala niezerowa czestotliwosc. Statusy `idle` i `no_samples`
oraz raportowanie brakow/zer sprawdzono regresjami harnessu/reportowania.

Hz zostaly przeliczone na MHz raz. Pamiec i bandwidth nie byly ponownie
skalowane przez harness; niezerowe stawki pokryto regresjami.
Pierwsze pola NPU odrzucono przy trzech probkach, bez usuwania calej probki
innych rodzin urzadzen. Agregacja zaokragla wartosci zgodnie z kodem harnessu.

UI NPU: 2 testy renderowania przeszly; sprawdzono jednokrotne Hz->MHz,
zachowanie utilization 0 i undefined oraz wskaznik N/A zamiast sztucznego zera.
TUI objete bylo pelna regresja Go z `-race`.

## Wykryta niespojnosc: drugi licznik bandwidth PTL

**Potwierdzona syntetycznie; skutku na fizycznym PMT nie zweryfikowano.**

[Kolektor Go](microservices/metrics-manager/native/internal/npu/pmt.go#L69)
definiuje dla PTL tylko `memoryBW: 0xc18`.
[Standalone](tools/npu-monitor-tool/npu-monitor-tool.py#L88)
definiuje `VPU_MEMORY_BW: [0xc18, 0xc20]` i sumuje oba liczniki.

### Reprodukcja

1. Utworzyc syntetyczny sysfs z PMT GUID `0x3086000`, pelnym buforem 4096 B
	 i urzadzeniem pod `bus/pci/drivers/intel_vpu/0000:00:0b.0`.
2. Ustawic liczniki `0xc18=0` i `0xc20=0` jako 64-bit little-endian words.
3. Uruchomic aktualny `mm-plugin-npu -sysfs-root <fixture> -interval 500ms`
	 i odebrac pierwsza linie, aby potwierdzic istnienie baseline.
4. Pozostawic `0xc18=0`, zmienic tylko `0xc20` na 10000 i odebrac druga linie.
5. Porownac delta z `get_noc_bandwidth()` standalone, uzywajac roznicy
	 timestampow obu linii jako elapsed time.

W wykonanej probie delta-t wynosila **0.499786405 s**:

- Standalone: **20.00854745138576 MB/s**.
- Kolektor Go: **`bandwidth=0`**.
- Dodatkowa asercja wykrywajaca utrate ruchu drugiego licznika zakonczyla sie
	oczekiwanym bledem. Nie jest uwzgledniona jako pass w tabeli regresji.

### Proponowana poprawka

Potwierdzic semantyke obu licznikow w schemacie PMT dla GUID `0x3086000`.
Jezeli oba skladaja sie na wymagany ruch NPU, ujednolicic kolektor Go
i standalone: odczytac oba, liczyc delta i wrap dla kazdego osobno,
nastepnie sumowac stawki w decimal MB/s. Kontrola dlugosci bufora musi
uwzgledniac wszystkie wymagane rejestry.

Dodac regresje ze zmiana tylko `0xc20`, niezaleznym wrap kazdego licznika
i truncated PMT obejmujacym pierwszy, ale nie drugi rejestr.
Nie dokonano tej poprawki, zgodnie z zakazem zmian kodu produkcyjnego.

## Niewykonane lub ograniczone kontrole

| Kontrola | Powod / zakres dostepnego dowodu |
| --- | --- |
| Fizyczny odczyt PLL, power, temperature i bandwidth PMT | `Permission denied` dla binarnego PMT |
| Fizyczny eksport `npu_frequency` w Hz i porownanie PLL ze sterownikiem | Kolektor nie mogl zainicjalizowac PMT; odczytano tylko MHz sterownika |
| Fizyczna sciezka collector -> Telegraf -> API -> harness | Zablokowana na PMT; pelna sciezka przeszla tylko na fixture |
| Rzeczywisty wplyw pomijania `0xc20` na bandwidth hosta | Brak uprawnien PMT i niezaleznego potwierdzenia semantyki tego licznika |
| Interaktywne UI w przegladarce i TUI z danymi sprzetowymi | Wykonano regresje renderowania/UI i Go/TUI, nie sesje interaktywne |
| Testy Docker/Compose uruchomionego stosu | Docker nie byl dostepny; nie instalowano go globalnie |
| 56 testow Helm | Helm CLI nie byl dostepny |
| Fizyczne testy MTL | Ten host jest PTL; MTL zweryfikowano syntetycznie |

Brakujacych kontroli nie oznaczono jako pass ani nie zastepowano braku danych
wartoscia zero. Uzyskanie fizycznego odczytu PMT wymaga uruchomienia przez
operatora w juz autoryzowanym kontekscie; nie zmieniano uprawnien w tej sesji.

## Sprzatanie i stan koncowy

Usunieto wylacznie utworzone przez testy srodowiska, fixture, skrypty,
cache, kopie UI i plik bytecode utworzony przez pierwszy odczyt standalone.
Katalog `/tmp/npu-validation-5QMDcMch` zostal usuniety w calosci.
Nie pozostaly wlasne procesy testowe. Nie tworzono kontenerow ani wolumenow.
Istniejace obrazy, wolumeny i uslugi nie byly usuwane ani restartowane.

Koncowe odczyty sysfs po testach:

```text
npu_busy_time_us=3928311
npu_memory_utilization=68714496
npu_current_frequency_mhz=0
npu_max_frequency_mhz=2050
power_state=D3hot
```

Busy-time zachowal skumulowany wynik inferencji; nie resetowano licznika.
Pamiec wrocila do 65.53125 MiB, NPU do idle.
Branch i HEAD pozostaly niezmienione, a repozytorium bylo czyste po
sprzataniu testow. Ten raport zapisano pozniej na polecenie uzytkownika;
przy jego zapisie nie uruchamiano ponownie testow sprzetowych.
