<!-- SPDX-FileCopyrightText: (C) 2026 Intel Corporation -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Ukierunkowany fizyczny retest NPU MTL - 2026-11-13

## Wynik

**PARTIAL PASS, nie pelny PASS.** Autoryzowany odczyt PMT i swiezy kolektor dzialaja fizycznie.
Jednostki czestotliwosci, limit MHz, pamiec MiB oraz wszystkie 109 porownywalnych delt
mocy i wykorzystania przeszly kontrole. Replay realnych danych przez Telegraf, biezace
API/SSE, harness i swieze TUI przeszedl. Dwie temperatury nie przeszly scislego
porownania z sasiednimi snapshotami PMT; przyczyna pozostaje **UNVERIFIED**.

Nie powtarzano pelnej regresji Go/pytest/UI. PTL, fizyczne wrap/reset i przegladarka UI
nie zostaly zweryfikowane. Nie zmieniono kodu produkcyjnego ani konfiguracji uslug.

Poprzedni [test-result-mtl.md](test-result-mtl.md) pozostawiono bez zmian.
Artefakty znajduja sie poza repozytorium Git:
[opis i odtwarzanie](../npu-mtl-retest-2026-11-13-data/README.md),
[analiza](../npu-mtl-retest-2026-11-13-data/data/analysis.json),
[manifest SHA256](../npu-mtl-retest-2026-11-13-data/manifest.sha256).

## Checkout I Host

| Element | Potwierdzony stan |
|---|---|
| Repozytorium | `/home/fst/okomor/edge-ai-libraries` |
| Branch | `test/npu-metrics`, bez przelaczania |
| HEAD | `ecad3438bb5715272e1356f85e4eac24b908dc13`, commit `mtl test` |
| Poczatkowy Git | Czysty; bez zmian tracked/staged |
| CPU | Intel Core Ultra 7 165H; family 6, model 170 / 0xAA, stepping 4, 22 CPU |
| NPU PCI | `0000:00:0b.0`, `8086:7d1d`, rev 04, Meteor Lake NPU |
| Kernel | `6.17.0-23-generic` |
| Driver | `intel_vpu`, `1.0.0 6.17.0-23-generic`, srcversion `7F053039ED3AE4C72B280DE` |
| PMT | `telem1`, GUID `0x130670b2`, size 3352, offset 744: **MTL** |
| Uprawnienia PMT | `root:root 0440`, bez zmiany |
| Urzadzenie inferencji | `/dev/accel/accel0`, bez zmiany uprawnien |
| Firmware SHA256 | `0c3dac41445653d1a8fc95b17d14366a982d112bfd1a6bb664b9ec763abcbfd8` |

Metadane CPU, PCI, drivera, firmware, wszystkich GUID-ow i uprawnien zachowano w katalogu danych.

## Dostep PMT I Swiezy Build

Odczyt jako UID 0 w izolowanym kontenerze udany. Uzyto `--cap-drop ALL`,
`--security-opt no-new-privileges`, `--network none` i bind `/sys` **read-only**.
Koncowy odczyt potwierdzil zerowe CapInh/Prm/Eff/Bnd/Amb i `NoNewPrivs=1`.
Nie uzyto sudo, `--privileged`, dodatkowych capabilities, chmod sysfs/urzadzen ani
zmian drivera, firmware, ustawien zasilania czy istniejacych uslug.

Z biezacego checkoutu zbudowano `mm-plugin-npu` i `mm-tui`, Go 1.26.0.
Oficjalne archiwum Go zweryfikowano SHA256 przed wykonaniem. Build info obu binarek:
`vcs.revision=ecad3438bb5715272e1356f85e4eac24b908dc13`, `vcs.modified=false`.
Wszystkie toolchain/cache/venv byly w `/tmp/npu-mtl-retest.S7r3ARsV`.

Swieze fizyczne `-once`: exit **0**, rozpoznanie `Meteor Lake`, brak `permission denied`:

```text
npu,host=f597134b4a9d power=0,frequency=0,temperature=35i,tile_config=0i,utilization=0i,memory_mb=65.5390625,power_state=3i,frequency_max_mhz=1400 1794561603929742055
```

Pierwszy launch mial exit 126 przed uruchomieniem binarki: mount calego prywatnego
katalogu temp 0700 nie byl przechodni dla root bez DAC override. Bezposredni mount
katalogu binarek rozwiazal problem, bez zmiany uprawnien. To nie byl blad odczytu PMT.
Wczesniejszy BLOCKED dla nieuprzywilejowanego czytelnika nie jest automatycznie
naprawiony konfiguracja produkcyjna: ten retest uzywa autoryzowanej tozsamosci root.

## Przebieg Fizyczny

Jeden przebieg, okolo 09:22:37-09:23:32 UTC. OpenVINO
`2026.5.0-23697-32776832f8d`, istniejacy obraz `intel/vippet-app:2026.3.0-dev`.
Jawne `core.compile_model(model, "NPU")`; bez AUTO/HETERO/CPU fallback.
`EXECUTION_DEVICES` normalizowane dla string/list; fizyczny wynik `['NPU.0']`.

| Faza | Czas i czynnosci |
|---|---|
| Idle | 10.000108 s; bez inferencji badanego modelu |
| Prepare | Oddzielne 0.088976 s: compile, alokacja i kontrolna inferencja |
| Load | **30.002427 s**, 49 471 inferencji; przekroczenie ok. 2.4 ms obejmuje koncowa kontrole |
| Zwolnienie modelu | Ok. 12 ms przed recovery; ujeto w zdarzeniach |
| Recovery | Oczekiwanie 15 s; do zakonczenia kolektora 15.003550 s |

Model FP32 `[1,512,512]`: `ReLU(MatMul(input, 0.5 * identity))`.
Naprzemienne wejscia o przeciwnych znakach. **52 kontrole wyniku PASS**,
`rtol=0.02`, `atol=0.002`: przed obciazeniem, co 1000 inferencji oraz na koncu.
Wszystkie wywolania inferencji zakonczyly sie bez bledu.

Kolektor zakonczyl sie exit **0**, stderr zawieral tylko rozpoznanie MTL z obsluga busy/memory.

## Surowe Dane I Czas

Zachowano **5232 rekordy**: 5112 niezaleznych odczytow raw, 110 linii native, 10 zdarzen.
Kazdy raw zawiera pelne 3352 B PMT w base64, timestampy wall/monotonic, poczatek/koniec
odczytu PMT oraz indywidualnie zmierzone odczyty busy, memory, max i frequency przed/po PMT.
Nie zastapiono bledow odczytu zerami.

| Parametr | Rzeczywisty wynik |
|---|---|
| Native, nominalnie 500 ms | Odstepy 0.499003-0.501141 s |
| Raw, oczekiwanie 10 ms po odczycie | Srednio 10.780 ms, zakres 10.438-21.201 ms |
| Najblizszy raw PMT do timestampu native | Srednio 2.801 ms, maksimum 5.393 ms |
| Caly pojedynczy odczyt raw | Maksimum 7.832 ms |
| Integralnosc snapshotow | 5112/5112: poprawny base64, dlugosc, ponowny decode |

Nie zakladano stalego czasu przy obliczaniu delt. Dla mocy i busy wykorzystano
raw przed/po obu timestampach native, ograniczajac mozliwa delte z niezaleznych licznikow.
Szerokosc takich przedzialow wyniosla maksymalnie 0.024524 W i 3 punkty procentowe.
Najwieksza roznica mocy wobec pojedynczej najblizszej pary raw: 0.008790 W.
Nie jest to pomiar kalibracyjny watomierzem ani dowod bezwzglednej dokladnosci czujnika.

## Jednostki I Delty

| Kontrola | Wynik |
|---|---|
| MTL PLL `0x68[7:0]`, `ratio * 50e6 * 2/3` | **110/110 PASS** native vs sasiednie PMT; ratio 42 = 1 400 000 000 Hz |
| Hz vs driver MHz | 5101/5104 raw z rownym odczytem drivera przed/po PMT zgodne po obcieciu PMT Hz do calych MHz |
| Limit max | **110/110 PASS**, 1400 MHz, bez mnozenia przez 1e6 |
| Memory | **110/110 PASS**, bajty / 1024 / 1024, nie decimal MB |
| Busy/utilization | **109/109 PASS** w niezaleznych granicach delt; realny elapsed, calkowity procent, 0-100 |
| Energy/power | **109/109 PASS** w niezaleznych granicach: `delta(raw / 2^14) / elapsed`, J/s = W |
| Temperatura `0x98[47:40]` | Decode raw zgodny; native rowne sasiednim raw **108/110**, dwa odstepstwa ponizej |
| Sensownosc temperatury | **110/110 PASS**, zakres native 35-43 stopni w -once/przebiegu |
| Ujemne/NaN/Inf | **110/110 PASS**, brak takich wyjsc |
| Kanoniczne pola native | **110/110 PASS**, osiem oczekiwanych nazw, bez duplikujacych aliasow jednostek |
| MTL bandwidth | **110/110 PASS**, pole nieobecne, nie zero |

Pierwsza delty-baseline NewCollector nie jest niezaleznie timestampowana w zapisie,
dlatego pierwszy punkt nie jest wlaczony do 109 porownan power/utilization.
Pierwszy punkt native po inicjalizacji mial rzeczywiste 700 MHz, 0.509614 W i 0% busy;
nie udawano w nim zerowego zegara/mocy. Kolejna pelna probka idle miala jeszcze 0.174139 W.

Trzy rozbieznosci raw PLL/driver wystapily przy usypianiu albo ramp-up:
`1794561758338153103` i `1794561798240267223` ns: PMT 700 MHz, driver 0 MHz;
`1794561767820786286` ns: PMT 766.667 MHz, driver 800 MHz.
Sasiednie odczyty potwierdzaja przejscia. Odczyty nie sa atomowe; nie traktuje sie
tych trzech punktow jako zgodnych ani jako dowodu bledu wzoru czestotliwosci.

Pamiec raw obejmowala 68 722 688 B (65.5390625 MiB), przejsciowo
81 506 304 B (77.734375 MiB) oraz 83 607 552 B (79.734375 MiB).
Podczas pelnego load utrzymywala 79.734375 MiB, po zwolnieniu modelu wrocila do 65.5390625 MiB.
Nie zaobserwowano spadku licznikow energy/busy ani fizycznego wrap/reset; niczego nie wymuszano.

## Temperatura: Pozostale Odstepstwa

| Indeks native, od zera | Timestamp ns | Native | Sasiednie raw PMT |
|---|---|---|---|
| 20, pierwszy mieszany przedzial load | `1794561768164535071` | 37 | 38 |
| 47, srodek load | `1794561781665320369` | 41 | 40 |

Roznica 1 stopnia; takze odczyty raw w lokalnym oknie ok. +/-30 ms nie pokazaly
wartosci native. Biezacy getter native i niezalezny decode raw odczytuja te same bity
`0x98[47:40]`. Bez snapshotu faktycznie uzytego wewnatrz kolektora nie da sie
rozstrzygnac przyczyny. Nie zaklada sie, ze jest to na pewno skutek niesynchronicznosci.
**Scisla kontrola rownosci: 2 FAIL; przyczyna: UNVERIFIED.** Analizator zachowal exit **1**;
nie poszerzono tolerancji, nie usunieto odstepstw ani nie poprawiano produkcji.

## Porownanie Faz

Statystyki tylko dla pelnych przedzialow w danej fazie: 18 idle, 59 load, 29 recovery.
Pierwszy punkt i trzy przedzialy przekraczajace granice pozostaja w surowym zapisie;
nie wlaczono ich do tej tabeli, lecz wszystkie porownywalne delty sprawdzono osobno.

| Pole | Idle | Load | Recovery |
|---|---|---|---|
| Frequency Hz | 0 | 1 400 000 000 | 0 |
| Power W, min-max; avg | 0-0.174139; 0.009674 | 0.525823-0.532251; 0.528329 | 0-0.076636; 0.002643 |
| Utilization %, min-max; avg | 0 | 61-67; 66.1695 | 0 |
| Temperature, min-max; avg | 36 | 38-43; 40.7119 | 38-41; 39.6552 |
| Memory MiB | 65.5390625 | 79.734375 | 65.5390625 |
| Max MHz | 1400 | 1400 | 1400 |

Pierwszy mieszany punkt load: 58% i 0.473513 W. Pierwszy mieszany recovery:
11%, 0.501635 W, 700 MHz, ale pamiec juz 65.5390625 MiB.
To okna obejmujace aktywnosc przed granica; nie wymagano natychmiastowej zerowej delty.

## Realne Dane W Konsumentach

**PASS: replay dziewieciu realnych probek**, po trzy z idle, srodka load i konca recovery.
Wartosci i oryginalne timestampy zachowano; nie generowano syntetycznych metryk.
Telegraf 1.40.3, git `41b45388`, z istniejacego obrazu `intel/metrics-manager:2026.3.0-rc1`.
API uruchomione z biezacego `app.main:app`, harness z biezacego `HardwareMonitor`.
Osobny venv i katalog custom-metrics; tylko wlasne porty loopback.

```text
native LP -> Telegraf -> API /metrics/stream -> harness
                    -> Telegraf /metrics -> swieze mm-tui
```

| Gate | Wynik |
|---|---|
| Telegraf/API SSE | 9/9: wszystkie osiem npu_* nazw, liczby i timestampy zrodlowe ms zachowane |
| Harness | 9/9: Hz -> MHz raz; power/util/temp/memory bez zmiany; bandwidth nie dodano |
| Statusy | Idle `idle`, load `ok`, recovery `idle` |
| Liczniki fazy | `sample_count=3`, `npu_sample_count=2`, `npu_settle_samples_dropped=1` |
| TUI | Trzy exit 0; dodatkowe kontrole liczbowe wszystkich trzech widokow PASS |
| Widok load | 1.40 GHz of 1.40 GHz, 66.0%, 0.53 W, 42.0 stopnia, 80 MiB po zaokragleniu |
| Widoki idle/recovery | 0 Hz, 0.0%, 0.00 W, 66 MiB po zaokragleniu; temperatura 36/38 |
| Brak bandwidth | Nieobecne w eksporcie/harnessie; TUI pokazuje brak danych, nie 0 |

Pierwsza proba replay wskazala TUI na API `/metrics`, ktore eksportuje metryki
aplikacji/custom, nie ten strumien NPU. TUI mialo exit 0, ale bez panelu NPU;
test setup FAIL zachowano. Po wskazaniu udokumentowanego endpointu Telegraf powyzsze
bramki przeszly bez zmian produkcji. To jest **replay**, nie live end-to-end retest.
UI/browser nie uruchamiano i nie uznaje sie go za zweryfikowane.

## PTL I Zakres Wnioskow

Ten host to **MTL, nie PTL**. PTL formula `ratio * 50e6 / 2`, fizyczne raw counters
`0xc18`/`0xc20` oraz porownanie ich ze standalone: **NOT APPLICABLE / UNVERIFIED** tutaj.
Nie interpretowano bajtow MTL pod tymi offsetami jako licznikow PTL.
Nie pozyskano niezaleznej dokumentacji rozstrzygajacej semantyke ich laczenia.
Roznica pomiedzy implementacjami native/standalone lub sama aktywnosc licznikow nie
udowadnia koniecznosci ich sumowania. Wczesniejszy syntetyczny wynik nie stanowi takiego
dowodu; nie proponuje sie tutaj naprawy sumy na jego podstawie.

Standalone fizycznie nie uruchamiano w tym retest; nie przenosi sie jego poprzednich
syntetycznych PASS na aktualna telemetrie. Brak fizycznych testow zaginionych pol,
reset/wrap, innych generacji oraz bezwzglednej kalibracji energii/temperatury.

## Artefakty I Zakonczenie

Archiwum ok. 2.3 MiB, sam pelny capture gzip ok. 1.8 MiB.
SHA256 nieskompresowanego `physical.jsonl`:
`d77832bee925db9170a99d8d4cf0719986f7055ea8d12c9b29c1a117377c8b2a`.
Zweryfikowano bezstratna kompresje oraz SHA256 kazdego zachowanego pliku.
Dokladne wersje obrazu/pakietow, build identities, skrypty, logi i failed setup sa w archiwum.

Wlasne procesy Telegraf/API zakonczono; wszystkie szesc uzytych portow bylo zamknietych.
Wlasne kontenery usuniete przez `--rm` albo jawne usuniecie pomocniczego kontenera kopii.
Nie pozostal zaden kontener. Usunieto tylko wlasny temp workspace z binarkami/toolchain,
venv i lokalnymi cache; nie czyszczono cache wspolnych, obrazow ani cudzych zasobow.
Przed usunieciem lokalnego Go cache nadano write tylko jego wlasnym katalogom UID 1000.

HEAD/branch pozostaly bez zmian, tracked i staged diff puste. Stare raporty MTL i PTL
przeszly SHA256 check bez zmian; inventory obrazow identyczne. Nie wykonywano commit/push.
Jedyny nowy plik w repozytorium to niniejszy osobny raport; surowe pomiary sa poza Git.