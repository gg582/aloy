# aloy

C++ 패키지 매니저 & 메타 빌드 시스템

**aloy**는 복잡한 CMakeLists.txt를 직접 작성하지 않고, YAML 설정 하나로 의존성 관리부터 빌드까지 한 번에 수행할 수 있게 돕는 Go 기반 도구입니다.

## 핵심 철학

- **CMake 추상화** — `project.yaml`만 관리하면 Modern CMake (Target-based)가 자동 생성됩니다
- **로컬 격리** — 의존성은 `.my_modules/`에 격리 저장하여 전역 환경 오염과 ABI 충돌을 방지합니다
- **버전 충돌 감지** — 동일 패키지에 대해 메이저 버전이 다르면 에러를 발생시키고 `alias` 사용을 권고합니다
- **레거시 호환** — 기존 CMakeLists.txt가 있는 일반 오픈소스 저장소도 그대로 사용할 수 있습니다

## 설치

```bash
go install github.com/snowmerak/aloy@latest
```

### 전제 조건

시스템 PATH에 다음 도구가 필요합니다:

- `git`
- `cmake` (>= 3.15)
- C++ 컴파일러 (GCC, Clang, MSVC 등)

## 빠른 시작

```bash
# 실행 파일 프로젝트 초기화
mkdir my_project && cd my_project
aloy init

# 라이브러리 프로젝트 초기화
aloy init --type library

# 의존성 추가
aloy add https://github.com/gabime/spdlog.git --cmake-option SPDLOG_BUILD_TESTS=OFF
aloy add https://github.com/nlohmann/json.git --cmake-target nlohmann_json
aloy add OpenSSL --system

# 동기화 (의존성 해석 + CMake 생성 + cmake configure)
aloy sync

# 빌드
aloy build
```

## 프로젝트 구조

```
my_project/
├── project.yaml             # 프로젝트 정의 및 의존성 명세
├── aloy.lock                # 고정된 의존성 버전 및 커밋 해시
├── src/                     # 소스 코드
├── include/                 # 공용 헤더
├── .my_modules/             # [자동] 다운로드된 패키지 소스
├── build/                   # [자동] CMake 빌드 결과물
└── CMakeLists.txt           # [자동] aloy가 생성한 마스터 스크립트
```

## project.yaml

```yaml
project:
  name: MyStreamingServer
  version: 1.0.0
  cxx_standard: 17

targets:
  mediaserver:
    type: executable           # executable, library, shared_library, header_only
    sources:
      - "src/main.cpp"
      - "src/core/**/*.cpp"    # glob 패턴 지원
    includes:
      public: ["include/"]
      private: ["src/core/private"]

    platforms:
      linux:
        sources: ["src/platform/linux/**/*.cpp"]
        compiler_flags: ["-O2", "-g"]
      windows:
        sources: ["src/platform/win/**/*.cpp"]
        compiler_flags: ["/wd4819"]

    dependencies:
      - name: logger
        git: "git@github.com:company/logger.git"
        version: "^1.2.0"
        alias: alog

      - name: OpenSSL
        type: system

      - name: spdlog
        git: "https://github.com/gabime/spdlog.git"
        cmake_options:
          SPDLOG_BUILD_TESTS: "OFF"

      - name: json
        git: "https://github.com/nlohmann/json.git"
        cmake_target: nlohmann_json    # CMake 타겟 이름이 패키지 이름과 다를 때

inject_cmake: "cmake/extra_logic.cmake"
```

### 의존성 필드

| 필드 | 필수 | 설명 |
|---|---|---|
| `name` | O | 패키지 이름 (Git URL에서 자동 추론 가능) |
| `git` | △ | Git 저장소 URL (system 타입이면 불필요) |
| `version` | - | SemVer 제약 조건 (`^1.2.0`, `~1.0`, `>=1.0.0 <2.0.0`) |
| `type` | - | `"system"` 이면 `find_package()` 사용 |
| `alias` | - | 별칭 — 디렉토리 이름 및 참조 이름으로 사용 |
| `cmake_target` | - | 링킹 시 사용할 CMake 타겟 이름 (패키지 이름과 다를 때) |
| `cmake_options` | - | `key: value` 맵 — `set(KEY VAL CACHE ... FORCE)` 로 주입 |

> **`cmake_target`은 언제 필요한가?**
> 일부 라이브러리는 CMake 타겟 이름이 저장소 이름과 다릅니다.
> 예: `nlohmann/json` 저장소의 CMake 타겟은 `nlohmann_json::nlohmann_json`이 아닌 `nlohmann_json`입니다.
> 이런 경우 `cmake_target: nlohmann_json`을 지정해야 `target_link_libraries`에서 올바르게 링킹됩니다.

## CLI 명령어

| 명령어 | 설명 |
|---|---|
| `aloy init` | 프로젝트 초기화 (`project.yaml`, `src/`, `include/` 생성) |
| `aloy add <url\|name>` | 의존성 추가 |
| `aloy remove <name>` | 의존성 제거 (이름 또는 별칭으로 검색) |
| `aloy sync` | 의존성 해석 → CMake 생성 → `cmake -B build` 실행 |
| `aloy build` | 빌드 수행 (필요 시 sync 자동 호출) |
| `aloy clean` | `build/` 삭제 (`--all`로 `.my_modules/` + `CMakeLists.txt`도 삭제) |
| `aloy download` | `aloy.lock` 기반으로 소스만 클론 (CI용) |
| `aloy update` | SemVer 범위 내 최신 버전으로 갱신 (`--dry-run` 지원) |

### init 옵션

```bash
aloy init                    # executable 프로젝트 (기본)
aloy init --type library     # 정적 라이브러리 프로젝트
aloy init -t shared_library  # 공유 라이브러리 프로젝트
aloy init -t header_only     # 헤더 온리 라이브러리 프로젝트
```

### add 옵션

```bash
aloy add <git_url>                              # 기본 추가
aloy add <git_url> -v "^1.2.0"                 # 버전 제약
aloy add <git_url> -a myalias                  # 별칭 지정
aloy add <name> --system                        # 시스템 패키지 (find_package)
aloy add <git_url> --cmake-option KEY=VAL       # CMake 옵션 전달 (반복 가능)
aloy add <git_url> --cmake-target target_name   # CMake 타겟 이름 지정
aloy add <git_url> -t mytarget                 # 특정 타겟에 추가
```

### build 옵션

```bash
aloy build                    # Release 빌드 (기본)
aloy build --config Debug     # Debug 빌드
aloy build -j 8               # 병렬 빌드
```

### clean 옵션

```bash
aloy clean                    # build/ 디렉토리만 삭제
aloy clean --all              # build/ + .my_modules/ + CMakeLists.txt 모두 삭제
```

### update 옵션

```bash
aloy update                   # 모든 의존성을 범위 내 최신으로 갱신
aloy update --dry-run         # 변경 사항만 출력하고 실제 갱신하지 않음
```

## 의존성 해석

aloy의 의존성 해석은 **BFS (너비 우선 탐색)** 기반으로 동작합니다:

1. 모든 타겟의 의존성을 큐에 넣고 순서대로 처리합니다
2. 각 의존성에 대해 Git 태그에서 SemVer 호환 버전을 탐색합니다 (`v1.2.0` → `1.2.0`)
3. `^1.2.0`, `~1.2.0`, `>=1.0.0 <2.0.0` 등 SemVer 범위를 지원합니다
4. 동일 패키지가 여러 번 요구되면 **먼저 해석된 버전을 유지**합니다 (first-come-first-served)
5. 단, **메이저 버전이 충돌**하면 에러를 발생시키고 `alias` 사용을 권고합니다
6. 태그가 없거나 매칭 태그가 없는 저장소는 기본 브랜치(`main` → `master`)로 폴백합니다
7. 의존성이 aloy 패키지(`project.yaml` 보유)이면 하위 의존성도 재귀적으로 해석합니다

### 전이 의존성과 `cmake_target`

aloy 서브패키지(`.my_modules/` 내에서 `project.yaml`을 가진 패키지)는 재귀적으로 의존성이 해석됩니다. 하지만 **서브패키지의 `cmake_target` 설정은 상위 프로젝트로 전파되지 않습니다.**

예를 들어:
- 패키지 A가 `nlohmann/json`에 의존하고 `cmake_target: nlohmann_json`을 설정했더라도
- A를 사용하는 상위 프로젝트에서 직접 `nlohmann/json`도 사용한다면, 상위 프로젝트에서도 `cmake_target: nlohmann_json`을 별도로 지정해야 합니다

이는 각 프로젝트의 `project.yaml`이 독립적으로 관리되기 때문입니다. 전이 의존성의 CMake 타겟 이름은 해당 패키지의 원래 CMakeLists.txt가 정의하는 타겟으로 해석됩니다.

## CMake 생성

- `file(GLOB)` 대신 소스 파일 목록을 명시적으로 하드코딩합니다
- `target_include_directories`, `target_link_libraries` 등 타겟 기반 명령어를 사용합니다
- 플랫폼별 `if(WIN32)` / `if(UNIX AND NOT APPLE)` / `if(APPLE)` / `if(UNIX)` 조건 블록을 생성합니다
- `type: system` 패키지는 `find_package()` 로 처리합니다
- `cmake_options`는 `set(VAR VAL CACHE STRING "" FORCE)` 로 주입합니다
- `cmake_target`이 지정된 의존성은 해당 이름으로 `target_link_libraries`에 링킹됩니다
- `inject_cmake` 로 커스텀 CMake 로직을 삽입할 수 있습니다
- aloy 서브패키지에 대해서는 `.my_modules/` 내에 별도의 CMakeLists.txt를 생성합니다

### 링킹 우선순위

`target_link_libraries`에서 사용되는 이름의 결정 우선순위:

1. `type: system` → 패키지의 `name`
2. `cmake_target`이 지정됨 → `cmake_target` 값
3. 그 외 → `alias`가 있으면 `alias`, 없으면 `name`

## 타겟 타입

| 타입 | CMake 매핑 | 설명 |
|---|---|---|
| `executable` | `add_executable()` | 실행 파일 |
| `library` | `add_library(STATIC)` | 정적 라이브러리 |
| `shared_library` | `add_library(SHARED)` | 공유 라이브러리 |
| `header_only` | `add_library(INTERFACE)` | 헤더 온리 라이브러리 (소스 불필요) |

## 알려진 제한 사항

- **순환 의존성 미감지** — 의존성 그래프에 순환이 있으면 무한 루프에 빠질 수 있습니다
- **SemVer 교집합 미지원** — 동일 패키지에 대해 서로 다른 버전 범위가 요구되면 먼저 해석된 버전이 사용됩니다 (교집합 계산 없음)
- **전이 의존성 cmake_target 미전파** — 위 "전이 의존성과 cmake_target" 섹션 참조

## 라이선스

MIT
