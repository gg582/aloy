#include <fmt/core.h>
#include <nlohmann/json.hpp>

int main() {
    nlohmann::json config = {
        {"name", "example"},
        {"version", 1},
        {"features", {"fmt", "json"}}
    };

    fmt::print("Config:\n{}\n", config.dump(2));

    for (const auto& feature : config["features"]) {
        fmt::print("  - {}\n", feature.get<std::string>());
    }

    return 0;
}