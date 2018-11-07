[![MacOS Build Status](https://circleci.com/gh/ethereumproject/go-ethereum/tree/master.svg?style=shield)](https://circleci.com/gh/ethereumproject/go-ethereum/tree/master)
[![Windows Build Status](https://ci.appveyor.com/api/projects/status/github/ethereumproject/go-ethereum?svg=true)](https://ci.appveyor.com/project/splix/go-ethereum)
[![Go Report Card](https://goreportcard.com/badge/github.com/ethereumproject/go-ethereum)](https://goreportcard.com/report/github.com/ethereumproject/go-ethereum)
[![API Reference](https://camo.githubusercontent.com/915b7be44ada53c290eb157634330494ebe3e30a/68747470733a2f2f676f646f632e6f72672f6769746875622e636f6d2f676f6c616e672f6764646f3f7374617475732e737667
)](https://godoc.org/github.com/ethereumproject/go-ethereum)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/ethereumproject/go-ethereum?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

## Ethereum Go (Ethereum Classic Blockchain)
체인을 지원하는 이더리움 프로토콜의 공식언어인 Go 언어로 구현을 진행합니다.
Ethereum Classic(ETC)은 개발자가 이더리움과 병행하여 강력한 응용 프로그램 플랫폼을 제공하면서 차별화 된 DAO 조치를 거부하고 있습니다.
## Install

### :rocket: Release Binary
가장 간단한 방법은 우리의 사이트인 [Releases page](https://github.com/ethereumproject/go-ethereum/releases)를 방문해서 압축파일을 다운로드 후 (사용자의 OS 환경 등에 맞게), 압축 해제된 파일을 사용자의 `$PATH`로 옮겨 줍니다. 그리고 터미널을 열어서 '$ geth help` 이 제대로 동작하는지 확인하니다. 추가 설치지침의 다음의 설치 위키를 확인하십시오. [Installation Wiki](https://github.com/ethereumproject/go-ethereum/wiki/Home#Developers).

#### :beers: Using Homebrew (OSX only)
```
$ brew install ethereumproject/classic/geth
```

### :hammer: Building the source

빌드의 중심에 있어야 할 때 설치를 진행하세요. 하지만 이 경우 정상적이지 않은 경우가 발생할 수 있으며 릴리스 이외의 버전에는 우선적으로 지원을 할 수 없습니다. 때문에 이 부분의 설치는 개발자에게만 권장합니다.

#### Dependencies
Building geth는 GO 언어의 1.9이상 버전과 C 컴파일러 그리고 리눅스 시스템에서
C 컴파일러의 사용이 가능해야 합니다. 예를들어 다음 명령어를 이용합니다. `sudo apt-get install
build-essential`. On Mac: `xcode-select --install`.

#### Get source and package dependencies
```
$ go get -v github.com/ethereumproject/go-ethereum/...`
```

#### Install and build command executables

원본에서 설치한 기본 실행파일은 다음의 경로에 기본적으로 설치됩니다. `$GOPATH/bin/`.

##### With go:

- the full suite of utilities:
```
$ go install github.com/ethereumproject/go-ethereum/cmd/...`
```

- just __geth__:
```
$ go install github.com/ethereumproject/go-ethereum/cmd/geth`
```

##### With make:
```
$ cd $GOPATH/src/github.com/ethereumproject/go-ethereum
```

- the full suite of utilities:
```
$ make install
```

- just __geth__:
```
$ make install_geth
```

> For further `make` information, use `make help` to see a list and description of available make
> commands.


##### Building a specific release
위의 모든 명령은 HEAD의 바이너리를 빌드할 떄 발생합니다. 특정 릴리스/태그를 사용하려면 설치하기전에 명시해야 합니다.

```shell
$ go get -d github.com/ethereumproject/go-ethereum/...
$ cd $GOPATH/src/github.com/ethereumproject/go-ethereum
$ git checkout <TAG OR REVISION>
# Use a go or make command above.
```

##### Using a release source code tarball
GO 디렉토리의 구조 때문에 tarball은 `$GOPATH`. 아래의 적절한 서브 디렉토리에 추출되어야 합니다.
아래의 명령은 v4.1.1 release 빌드를 진행하는 예제입니다.:

```shell
$ mkdir -p $GOPATH/src/github.com/ethereumproject
$ cd $GOPATH/src/github.com/ethereumproject
$ tar xzf /path/to/go-ethereum-4.1.1.tar.gz
$ mv go-ethereum-4.1.1 go-ethereum
$ cd go-ethereum
# Use a go or make command above.
```

## Executables

이 저장소는 'cmd' 디렉토리에있는 여러 wrappers/executables 파일을 포함합니다.

| Command    | Description |
|:----------:|-------------|
| **`geth`** | 메인 Ethereum CLI 클라이언트 입니다. 이것은 전체 노드(default) 보관 노드(모든 기록 상태 유지) 또는 경량 노드(데이터 검색 진행)로 실행 될 수 있는 이더리움 네트워크(주 네트워크 및 테스트 네트워크, 사설 네트워크)의 진입 점입니다. HTTP,WebSocket 및 IPC 전송위에 노출된 JSON RPC 종점을 통해 이더리움 네트워크에 대한 게이트웨이로 다른 프로세스에서 사용할 수 있습니다. 자세한 내용은 [Command Line Options](https://github.com/ethereumproject/go-ethereum/wiki/Command-Line-Options) 해당 위키페이지를 참조하십시오. |
| `abigen` | 이더리움 계약 정의를 사용하기 쉽고 컴파일 할 때 사용할 수 있는 유형으로 안전하게 GO 패키지로 변환하는 소스코드 생성기를 말합니다. 여기서 contract 바이트 코드를 사용할 수 있다면 확장 된 기능을 가진 [Ethereum contract ABIs](https://github.com/ethereumproject/wiki/wiki/Ethereum-Contract-ABI)에서 작동합니다. 그러나 Solidity 소스 파일도 받아들이므로 개발 과정이 훨씬 간소화 되었습니다. 자세한 내용은 다음 위키 페이지를 참조하십시오. [Native DApps](https://github.com/ethereumproject/go-ethereum/wiki/Native-DApps-in-Go). |
| `bootnode` | 이더리움 클라이언트 구현의 버전을 없애고 네트워크 노드 탐색 프로토콜에만 참여하지만 더 높은 수준의 응용 프로그램 프로토콜은 실행하지 않습니다. 가벼운 부트 스트랩 노드로 사설 네트워크에서 피어를 찾는데 더 도움이 될 수 있습니다. |
| `disasm` | EVM 바이트 코드를 보다 사용자 친화적인 어셈블리어와 같은 opcode로 변환하는 바이트 코드 디어셈블리어이다. (예. `echo "6001" | disasm`). For details on the individual opcodes, please see pages 22-30 of the [Ethereum Yellow Paper](http://gavwood.com/paper.pdf). |
| `evm` | 구성 가능한 환경 및 실행 모드 내에서 바이트 코드 스 니펫을 실행할 수 있는 EVM의 개발자 유틸리티 버전입니다. 그 목적은 EVM opcode의 insoltaed fine graned 디버깅을 허용하는 것입니다. (예. `evm --code 60ff60ff --debug`). |
| `gethrpctest` | [Ethereum JSON RPC](https://github.com/ethereumproject/wiki/wiki/JSON-RPC) 의 사양에 대한 기준 준수 여부를 검증하는 [ethereum/rpc-test](https://github.com/ethereumproject/rpc-tests) 테스트 슈트를 지원하는 개발자 유틸리티 도구를 말합니다. 자세한 내용은 다음을 참조하시오.[테스트 스위트의 readme](https://github.com/ethereumproject/rpc-tests/blob/master/README.md). |
| `rlpdump` | 바이너리 RLP ([Recursive Length Prefix](https://github.com/ethereumproject/wiki/wiki/RLP)) 덤프 (이더리움 프로토콜에 의해 사용되는 데이터 인코딩으로 네트워크는 물론 컨센서스가 확실하다.) 를 사용자 친환적으로 표현하도록 변환하는 개발자 유틸리티 도구 (e.g. `rlpdump --hex CE0183FFFFFFC4C304050583616263`). |

## :green_book: Geth: the basics

### Data directory
기본적으로 geth는 운영체제에 따라 상위 디렉토리에 모든 노드 및 블록 체인 데이터를 저장합니다.:

- Linux: `$HOME/.ethereum-classic/`
- Mac: `$HOME/Library/EthereumClassic/`
- Windows: `$HOME/AppData/Roaming/EthereumClassic/`

__다음과같이 디렉토리 지정이 가능합니다.__ with `--data-dir=$HOME/id/rather/put/it/here`.

상위 디렉토리 내에서 geth는 서브 디렉토리를 사용하여 실행하는 각 네트워크에 대한 데이터를 보유합니다. 기본 값은 다음과 같습니다.:

 - `/mainnet` for the Mainnet
 - `/morden` for the Morden Testnet

__하위 디렉토리를 지정할 수 있습니다. `--chain=mycustomnet`.

> __Migrating__: 만약 [3.4 Release](https://github.com/ethereumproject/go-ethereum/releases), 이전 데이터가 존재한다면 기존 표준 ETC 데이터가 마이그레이션을 진행합니다. 마이그레이션 관리에 대한 자세한 내용은 [3.4 릴리스 정보를 참조하십시오.](https://github.com/ethereumproject/go-ethereum/wiki/Release-3.4.0-Notes).

### 기본 이더리움 네트워크의 전체 노드

```
$ geth
```

ETC 블록체인 다운로드를 진행하기위해 ("sync")를 이용하면 전체 블록 다운로드 및 체인 노드가 설정됩니다. __However__, 그러나 이 경우 ol 'geth' 앞서 가서 다음 섹션을 검토하는 것이 좋습니다....

#### :speedboat: `--fast`


가장 일반적인 시나리오는 Ethereum Classic 네트워크와 간단하게 상호 작용하려는 사용자입니다. 자금 이체; 계약을 전개하고 상호 작용하며, 나의 것. 이 특정 유스 케이스의 경우, 사용자는 오래된 이력 데이터를 신경 쓰지 않으므로 네트워크의 현재 상태로 빠르게 동기화 할 수 있습니다.:
```
$ geth --fast
```

빠른 동기화 모드에서 geth를 사용하면 대량의 트랜잭션 레코드가 남지 않는 블록 상태 데이터 만 다운로드되므로 CPU 및 메모리를 많이 사용하지 않아도됩니다.

빠른 동기화는 자동으로 다음과 같은 경우에 __중지 됩니다__ (및 전제 동기화 사용):

- 체인 데이터베이스에 *어떤* 전체 블록이 들어있을 때
- 노드가 네트워크 블록 체인의 현재 헤드까지 동기화되었습니다.

`--mine` 과 `--fast`를 함께 사용하는 경우 geth는 설명한대로 작동합니다. 고속 모드에서 헤드까지 동기화 한 다음 체인 헤드에서 첫 번째 전체 블록을 동기화 한 후 마이닝을 시작합니다.

*Note:* To geth의 성능을 추가로 높이기 위해서는, `--cache=2054` 를 사용하여 데이터 베이스의 메모리 허용량을 늘려야 합니다. (예. 2054MB) 특히 HDD 사용자의 경우 동기화 시간을 크게 향상시킬 수 있습니다. 이 플래그는 선택 사항으로 1GB - 2GB 범위를 권장하지만 원하는대로 높이거나 낮게 설정할 수 있습니다.

### 계정 생성과 관리

Geth는 개인 키(암호화된 키) 파일을 생성 가져오기, 업데이트, 잠금 해제 및 관리할 수 있습니다. 키 파일은 JSON 형식이며 기본적으로 해당 체인 폴더의 `/keystore` 디렉토리; 에 저장됩니다. `--keystore` 플래그를 사용하여 사용자 정의 위치를 지정할 수 있습니다.

```
$ geth account new
```

이 명령은 새 계정을 만들고 계정을 보호하기 위해 암호를 입력하라는 메시지를 표시합니다.

다른 `계정` 하위 명령 포함 목록:
```
SUBCOMMANDS:

        list    print account addresses
        new     create a new account
        update  update an existing account
        import  import a private key into a new account

```

[Accounts Wiki Page](https://github.com/ethereumproject/go-ethereum/wiki/Managing-Accounts) 페이지에서 자세히 확인 가능합니다. 만약 (~100,000+) 의 많은 계정을 관리하고 싶다면, 다음 사이트에 방문하십시오. [Indexing Accounts Wiki page](https://github.com/ethereumproject/go-ethereum/wiki/Indexing-Accounts).


### Javascript 콘솔과 상호 작용
```
$ geth console
```

이 명령은Geth의 내장된 대화식 [JavaScript console](https://github.com/ethereumproject/go-ethereum/wiki/JavaScript-Console)을 시작합니다, 이 콘솔을 통해 [`web3` methods](https://github.com/ethereumproject/wiki/wiki/JavaScript-API) 뿐아니라 [management APIs](https://github.com/ethereumproject/go-ethereum/wiki/Management-APIs)까지 가능합니다. 이 부분은 모두 선택사항이며 이미 실행중인 Geth 인스턴스에 'geth attch'로 첨부할 수 있습니다.

좀 더 알고싶다면... [Javascript Console 위키 페이지](https://github.com/ethereumproject/go-ethereum/wiki/JavaScript-Console).


### 추가 정보

명령행 옵션의 전체 목록은 [CLI Wiki page](https://github.com/ethereumproject/go-ethereum/wiki/Command-Line-Options).

## :orange_book: Geth: 개발 및 고급 사용

### Morden Testnet
만약 이더리움 계약을 만들면서 즐기고 싶다면
스템 전체가 중단 될 때까지 실질적인 비용을 들이지 않고도 할 수 있습니다. 즉, 주 네트워크에 연결하는 대신 주 네트워크와 완전히 동등하지만 플레이 - 이더 만 사용하여 노드와 테스트 네트워크에 합류하려고합니다.
```
$ geth --chain=morden --fast console
```

`--fast` 플래그와 `console` 부속명령은 위와 완전히 동일한 의미를 가지며 testnet에서도 똑같이 유용합니다. 여기로 건너 뛴 경우 설명을 보려면 위를 참조하십시오.

`--chain=morden` 플래글르 지정하면 Geth 인스턴스가 재구성 됩니다:

 - 위에서 언급했듯이 Geth는 testnet데이터를 `morden` 하위폴더 (`~/.ethereum-classic/morden`)에 호스트 합니다.
 - 기본 이더리움 네트워크를 연결하는 대신 클라이언트는 다른 P2P 부트 노드, 다른 네트워크 ID 및 기성 상태를 사용하는 테스트 네트워크에 연결합니다

선택적으로 `--testnet` or `--chain=testnet` 을 사용할 수도 있습니다.

> *Note: 거래가 주 네트워크와 테스트 네트워크 (다른 시작 넌스) 사이를 가로 지르는 것을 방지하기위한 내부 보호 조치가 있지만, 항상 플레이 머니와 리얼 머니에 대해 별도의 계정을 사용해야합니다. 수동으로 계정을 이동하지 않는 한 Geth는 기본적으로 두 네트워크를 올바르게 분리하고 두 네트워크간에 계정을 제공하지 않습니다.

### 프로그래밍 방식으로 Geth 노드 인터페이스하기

개발자로서, 나중에는 Geth와 Ethereum 네트워크와 직접 상호 작용하기를 원할 것입니다. 이를 돕기 위해 Geth는 JSON-RPC 바탕의 APIs ([standard APIs](https://github.com/ethereumproject/wiki/wiki/JSON-RPC) 과
[Geth specific APIs](https://github.com/ethereumproject/go-ethereum/wiki/Management-APIs)). HTTP, WebSockets 및 IPC (유닉스 기반 플랫폼에서는 유닉스 소켓, Windows에서는 명명 된 파이프)를 통해 노출 될 수 있습니다.


IPC 인터페이스는 기본적으로 활성화되어 있으며 Geth에서 지원하는 모든 API를 제공하지만 HTTP 및 WS 인터페이스는 수동으로 활성화해야하며 보안상의 이유로 일부 API 만 노출해야합니다. 이것들은 켜거나 끌 수 있으며 예상대로 구성 할 수 있습니다.

HTTP based JSON-RPC API options:

  * `--rpc` Enable the HTTP-RPC server
  * `--rpc-addr` HTTP-RPC server listening interface (default: "localhost")
  * `--rpc-port` HTTP-RPC server listening port (default: 8545)
  * `--rpc-api` API's offered over the HTTP-RPC interface (default: "eth,net,web3")
  * `--rpc-cors-domain` Comma separated list of domains from which to accept cross origin requests (browser enforced)
  * `--ws` Enable the WS-RPC server
  * `--ws-addr` WS-RPC server listening interface (default: "localhost")
  * `--ws-port` WS-RPC server listening port (default: 8546)
  * `--ws-api` API's offered over the WS-RPC interface (default: "eth,net,web3")
  * `--ws-origins` Origins from which to accept websockets requests
  * `--ipc-disable` Disable the IPC-RPC server
  * `--ipc-api` API's offered over the IPC-RPC interface (default: "admin,debug,eth,miner,net,personal,shh,txpool,web3")
  * `--ipc-path` Filename for IPC socket/pipe within the datadir (explicit paths escape it)


위의 플래그로 구성된 Geth 노드에 HTTP, WS 또는 IPC를 통해 연결하려면 자체 프로그래밍 환경의 기능 (라이브러리, 도구 등)을 사용해야하며 모든 전송에서 [JSON-RPC](http://www.jsonrpc.org/specification)를 사용해야합니다. 여러 요청에 대해 동일한 연결을 재사용 할 수 있습니다!

> Note:그렇게하기 전에 HTTP / WS 기반 전송을 여는 보안 의미를 이해하십시오! 인터넷상의 해커들은 노출 된 API로 Ethereum 노드를 파괴하려고합니다! 또한 모든 브라우저 탭에서 로컬로 실행되는 웹 서버에 액세스 할 수 있으므로 악의적 인 웹 페이지가 로컬로 사용 가능한 API를 파괴 할 수 있습니다!

### 개인 / 사용자 정의 네트워크 작동

[Geth 3.4](https://github.com/ethereumproject/go-ethereum/releases) 부터는 외부 포스 구성 JSON 파일을 지정하여 사설 체인을 구성 할 수 있습니다. 여기에는 프로토콜 포크, 부트 노드 및 chainID에 대한 기능 구성뿐만 아니라 필요한 창시 블록 데이터가 포함됩니다.

[이 저장소의 / config 하위 디렉토리에있는 Mainnet 및 Morden Testnet 사양을 나타내는](). 전체 예제 외부 구성 파일을 찾으십시오. 이러한 파일 중 하나를 사용자 지정의 시작 지점으로 사용할 수 있습니다.


개인 네트워크에서는 모든 노드가 호환 가능한 체인을 사용하는 것이 중요합니다. 사용자 정의 체인 구성의 경우 체인 구성 파일 (chain.json)은 각 노드에 대해 동일해야합니다.

#### 외부 체인 구성 정의

외부 체인 구성 파일을 지정하면 기성 상태를 비롯하여 부트 노드 및 포크 기반 프로토콜 업그레이드를 통해 사용자 정의 블록 체인 / 네트워크 구성을 세부적으로 제어 할 수 있습니다.

```shell
$ geth --chain=morden dump-chain-config <datadir>/customnet/chain.json
$ sed s/mainnet/customnet/ <datadir>/customnet/chain.json
$ vi <datadir>/customnet/chain.json # make your custom edits
$ geth --chain=customnet [--flags] [command]
```

외부 체인 구성 파일은 다음 최상위 필드에 대한 유효한 설정을 지정합니다.

| JSON Key | Notes |
| --- | --- |
| `chainID` |  체인 신원. 필요한 chain.json이있는 chain 데이터에 대해 local / subdir을 결정합니다. 필수이지만 각 노드마다 동일하지 않아야합니다. 이것은 EIP-155에 도입 된 chainID 검증이 아니라는 점에 유의하십시오. 그 안에있는 protocal 업그레이드로 구성됩니다. `forks.features`. |
| `name` | _선택 과목. 사람이 읽을 수있는 이름, 즉 Ethereum Classic Mainnet, Morden Testnet._ |
| `state.startingNonce` | _선택 과목. 커스텀 논스로 상태 db를 초기화하십시오.. |
| `network` | 유효한 피어를 식별하기 위해 네트워크 ID를 결정합니다. |
| `consensus` | _선택 과목. "ethash"또는 "ethast-test"(개발 용) 중 사용할 작업 증명 알고리즘 |
| `genesis` | 기생 상태를 결정합니다. 노드를 처음 실행하면 기원 블록을 작성합니다. 다른 창 블록을 사용하여 기존 체인 데이터베이스를 구성하면 해당 노드를 덮어 씁니다. |
| `chainConfig` | 포크 기반 프로토콜 업그레이드, 즉 EIP-150, EIP-155, EIP-160, ECIP-1010 등의 구성을 결정합니다. 하위 키는 `forks` 및 `badHashes`입니다. |
| `bootstrap` | _선택과목 [enode format](https://github.com/ethereumproject/wiki/wiki/enode-url-format)부트 스트랩 노드를 결정합니다.. |
| `include` | 선택 과목. 포함 할 기타 구성 파일. 경로는 상대 경로가 될 수 있습니다 ('include' 필드가있는 구성 파일 또는 절대 경로). 구성 파일 각각은 "기본"구성과 동일한 구조를가집니다. 포함 된 파일은 배열에 지정된 것과 동일한 순서로 "기본"구성 뒤에 처리됩니다. 나중에 처리 된 값은 이전에 정의 된 값을 덮어 씁니다.

*필드 이름, state.startingNonce 및 합의는 선택 사항입니다. 필요한 필드가 누락되었거나 잘못되었거나 다른 플래그와 충돌하면 Geth가 당황 할 것입니다. 이렇게하면 --chain이 --testnet과 호환되지 않습니다. --data-dir과 호환됩니다.

외부 체인 구성에 대해 자세히 알려면, [External Command Line Options Wiki page](https://github.com/ethereumproject/go-ethereum/wiki/Command-Line-Options)에 방문하십시오.

##### 랑데뷰 포인트 생성

모든 참여 노드가 원하는 기원 상태로 초기화되면 다른 사람이 네트워크 및 / 또는 인터넷에서 서로 찾을 수있는 부트 스트랩 노드를 시작해야합니다. 깨끗한 방법은 전용 bootnode를 구성하고 실행하는 것입니다.

```
$ bootnode --genkey=boot.key
$ bootnode --nodekey=boot.key
```


bootnode를 온라인으로 설정하면 다른 노드가 연결하여 피어 정보를 교환하는 데 사용할 수있는 enode URL을 표시합니다. 표시되는 IP 주소 정보 (대부분 [:])를 외부 액세스 가능 IP로 바꾸어 실제 enode URL을 가져 오십시오.

*Note: 완전한 본격적인 Geth 노드를 부트 노드로 사용할 수 있지만 덜 권장되는 방법입니다..*

enodes 및 enode 형식에 대한 자세한 내용에 대해 알고싶으면 다음 사이트를 방문하시오 [Enode Wiki page](https://github.com/ethereumproject/wiki/wiki/enode-url-format).

##### 멤버 노드 시작하기

bootnode가 작동 가능하고 외부에서 연결할 수 있으면 (실제로 연결 되었는지 확인을 위해 다음을 시도 할 수 있다. `telnet <ip> <port>` ),  `--bootnodes` 플래그를 통해 피어 검색을 위해 부트 노드를 가리키는 모든 후속 Geth 노드를 시작합니다. 개인 네트워크 데이터를 기본값과 분리 된 상태로 유지하는 것이 바람직합니다. 이렇게하려면 커스텀`--datadir` 및 / 또는`--chain` 플래그를 지정하십시오

```
$ geth --datadir=path/to/custom/data/folder \
       --chain=kittynet \
       --bootnodes=<bootnode-enode-url-from-above>
```

*Note: 네트워크가 기본 및 테스트 네트워크와 완전히 분리되므로 트랜잭션을 처리하고 새 블록을 생성하도록 광부를 구성해야합니다.

#### 개인 마이닝


공용 Ethereum 네트워크의 마이닝은 OpenCL 또는 CUDA를 지원하는 ethminer 인스턴스가 필요한 GPU를 사용하는 경우에만 가능하기 때문에 복잡한 작업입니다. 그러한 설정에 대한 정보는 [EtherMining subreddit](https://www.reddit.com/r/EtherMining/) 과 [Genoil miner](https://github.com/Genoil/cpp-ethereum) repository를 통해서 얻을 수 있습니다.

그러나 사설망 설정의 경우, 단일 CPU 광구 인스턴스는 무거운 리소스를 필요로하지 않고 올바른 간격으로 블록의 안정적인 스트림을 생성 할 수 있기 때문에 실용적인 목적으로는 충분합니다 (단일 스레드에서 실행, 다중 인스턴스는 필요 없음) . 마이닝을위한 Geth 인스턴스를 시작하려면 다음과 같이 확장 된 모든 일반 플래그를 사용하여 실행하십시오.

```
$ geth <usual-flags> --mine --minerthreads=1 --etherbase=0x0000000000000000000000000000000000000000
```


단일 CPU 스레드에서 블록 및 트랜잭션 마이닝을 시작하고 모든 절차를 '--etherbase'로 지정된 계정으로 지정합니다. 기본 가스 제한 블록을 ( '--targetgaslimit') 수렴으로 변경하고 가격 거래를 ( '- 가스 요금')에서 허용하여 광업을 더 조정할 수 있습니다.

계정 관리에 대한 추가 정보는 다음 사이트를 참조하십시오 [Managing Accounts Wiki page](https://github.com/ethereumproject/go-ethereum/wiki/Managing-Accounts).


## Contribution


소스 코드를 도와 주신 것에 대해 감사드립니다.

민주적 인 참여, 투명성 및 성실성의 핵심 가치는 우리와 함께 깊게 펼쳐집니다. 우리는 모든 사람들의 공헌을 환영하며, 수정 사항이 적은 경우에도 감사를드립니다.  :clap:


이 프로젝트는 현재 어려운 프로젝트 인 [Ethereum (ETHF) Github project](https://github.com/ethereum) 에서 마이그레이션되었으며 프로젝트를 유지 관리하는 데 필요한 인프라를 단계적으로 마이그레이션해야합니다.

작업에 기여하고 싶다면, 메인 코드베이스를 검토하고 병합하기 위해 유지 보수자를위한 요청을 포크하고 수정하고 커밋하고 보내십시오. 좀더 복잡한 변경사항을 제출하려면 [our Slack channel (#development)](http://ethereumclassic.herokuapp.com/) 또는 [our Discord channel (#development)](https://discord.gg/wpwSGWn) 의 핵심개발자에게 먼저 확인하십시오. 이러한 변화가 프로젝트의 일반적인 철학과 일치하는지 확인하고 / 또는 조기 피드백을 통해 귀하의 노력을 훨씬 가볍게 만들뿐만 아니라 검토 및 병합 절차를 빠르고 간단하게 수행 할 수 있습니다..


환경 구성, 프로젝트 종속성 관리 및 절차 테스트에 대한 자세한 내용은 [Wiki](https://github.com/ethereumproject/go-ethereum/wiki) 를 참조하십시오.

## License

The go-ethereum library (i.e. cmd디렉토리의 모든 코드) [GNU Lesser General Public License v3.0](http://www.gnu.org/licenses/lgpl-3.0.en.html), 에따라 사용 허가되며 파일의 저장소에 the `COPYING.LESSER` 파일이 포함되어 있습니다.

The go-ethereum binaries (i.e. cmd 디렉토리의 모든 코드)는 [GNU General Public License v3.0](http://www.gnu.org/licenses/gpl-3.0.en.html), 따라 사용이 허가되며 COPIYING 파일의 Google 저장소에도 포함되어 있습니다.
