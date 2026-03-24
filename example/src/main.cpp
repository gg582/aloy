#include <fmt/core.h>
#include <nlohmann/json.hpp>
#include <testlib.h>

int main() {
    nlohmann::json config = {
        {"name", "example"},
        {"version", 1},
        {"features", {"fmt", "json", "testlib"}}
    };

    fmt::print("Config:\n{}\n", config.dump(2));

    for (const auto& feature : config["features"]) {
        fmt::print("  - {}\n", feature.get<std::string>());
    }

    fmt::print("\ntestlib::add(3, 4) = {}\n", testlib::add(3, 4));
    fmt::print("testlib::multiply(3, 4) = {}\n", testlib::multiply(3, 4));

    return 0;
}