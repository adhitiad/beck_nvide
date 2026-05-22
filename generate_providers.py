import re

def extract_set(content, pattern, set_name):
    matches = re.findall(pattern, content)
    unique_matches = sorted(list(set(matches)))
    
    output = f"// {set_name} groups all related dependencies\n"
    output += f"var {set_name} = wire.NewSet(\n"
    for match in unique_matches:
        output += f"\t{match},\n"
    output += ")\n\n"
    return output

with open("main.go", "r", encoding="utf-8") as f:
    content = f.read()

repo_pattern = r"(repository\.New[A-Za-z0-9_]+)"
repo_output = extract_set(content, repo_pattern, "RepositorySet")

usecase_pattern = r"(usecase\.New[A-Za-z0-9_]+)"
usecase_output = extract_set(content, usecase_pattern, "UseCaseSet")

handler_pattern = r"(delivery\.New[A-Za-z0-9_]+)"
handler_output = extract_set(content, handler_pattern, "HandlerSet")

with open("cmd/server/providers_gen.go", "w", encoding="utf-8") as f:
    f.write("package server\n\nimport (\n\t\"github.com/google/wire\"\n\t\"nvide-live/internal/repository\"\n\t\"nvide-live/internal/usecase\"\n\t\"nvide-live/internal/delivery\"\n)\n\n")
    f.write(repo_output)
    f.write(usecase_output)
    f.write(handler_output)
