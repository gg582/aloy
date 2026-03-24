# aloy

C++ 패키지 매니저 & 메타 빌드 시스템

**aloy**는 복잡한 CMakeLists.txt를 직접 작성하지 않고, YAML 설정 하나로 의존성 관리부터 빌드까지 한 번에 수행할 수 있게 돕는 Go 기반 도구입니다.

## 핵심 철학

- **CMake 추상화** — `project.yaml`만 관리하면 Modern CMake (Target-based)가 자동 생성됩니다
- **로컬 격리** — 의존성은 `.my_modules/`에 격리 저장하여 전역 환경 오염과 ABI 충돌을 방지합니다
- **단일 버전 합의** — 다이아몬드 의존성 발생 시 SemVer 교집합으로 하나의 버전을 선택합니다
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
# 프로젝트 초기화
mkdir my_project && cd my_project
aloy init

# 의존성 추가
aloy add https://github.com/gabime/spdlog.git --cmake-option SPDLOG_BUILD_TESTS=OFF
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

inject_cmake: "cmake/extra_logic.cmake"
```

## CLI 명령어

| 명령어 | 설명 |
|---|---|
| `aloy init` | 프로젝트 초기화 (`project.yaml`, `src/`, `include/` 생성) |
| `aloy add <url>` | 의존성 추가 |
| `aloy remove <name>` | 의존성 제거 |
| `aloy sync` | 의존성 해석 → CMake 생성 → `cmake -B build` 실행 |
| `aloy build` | 빌드 수행 (필요 시 sync 자동 호출) |
| `aloy clean` | `build/` 삭제 (`--all`로 `.my_modules/`도 삭제) |
| `aloy download` | `aloy.lock` 기반으로 소스만 클론 (CI용) |
| `aloy update` | SemVer 범위 내 최신 버전으로 갱신 (`--dry-run` 지원) |

### add 옵션

```bash
aloy add <git_url>                          # 기본 추가
aloy add <git_url> -v "^1.2.0"             # 버전 제약
aloy add <git_url> -a myalias              # 별칭 지정
aloy add <name> --system                    # 시스템 패키지 (find_package)
aloy add <git_url> --cmake-option KEY=VAL   # CMake 옵션 전달
aloy add <git_url> -t mytarget             # 특정 타겟에 추가
```

### build 옵션

```bash
aloy build                    # Release 빌드 (기본)
aloy build --config Debug     # Debug 빌드
aloy build -j 8               # 병렬 빌드
```

## 의존성 해석

- Git 태그에서 SemVer 호환 버전을 탐색합니다 (`v1.2.0` → `1.2.0`)
- `^1.2.0`, `~1.2.0`, `>=1.0.0 <2.0.0` 등 SemVer 범위를 지원합니다
- 동일 패키지가 여러 버전으로 요구되면 교집합 중 최고 버전을 선택합니다
- 메이저 버전 충돌 시 에러를 발생시키고 `alias` 사용을 권고합니다
- 태그가 없는 저장소는 `main` → `master` 순으로 폴백합니다

## CMake 생성

- `file(GLOB)` 대신 소스 파일 목록을 명시적으로 하드코딩합니다
- `target_include_directories`, `target_link_libraries` 등 타겟 기반 명령어를 사용합니다
- 플랫폼별 `if(WIN32)` / `if(UNIX AND NOT APPLE)` / `if(APPLE)` 조건 블록을 생성합니다
- `type: system` 패키지는 `find_package()` 로 처리합니다
- `cmake_options`는 `set(VAR VAL CACHE ... FORCE)` 로 주입합니다
- `inject_cmake` 로 커스텀 CMake 로직을 삽입할 수 있습니다

## 타겟 타입

| 타입 | CMake 매핑 |
|---|---|
| `executable` | `add_executable()` |
| `library` | `add_library(STATIC)` |
| `shared_library` | `add_library(SHARED)` |
| `header_only` | `add_library(INTERFACE)` |

## 라이선스

MIT
