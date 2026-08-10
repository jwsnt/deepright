from fastmcp import FastMCP

mcp = FastMCP("Dynamic Resource")

@mcp.resource("config://app-version")
def provide_app_version() -> str:
    """Returns the application version."""
    return "v2.1.0"

# Dynamic resource template expecting a 'user_id' from the URI
@mcp.resource("db://users/{user_id}/email")
async def get_user_email(user_id: str) -> str:
    """Retrieves the email address for a given user ID."""
    # Replace with actual database lookup
    emails = {"123": "alice@example.com", "456": "bob@example.com"}
    return emails.get(user_id, "not_found@example.com")

# Resource returning JSON data
@mcp.resource("data://product-categories")
def get_categories() -> list[str]:
    """Returns a list of available product categories."""
    return ["Electronics", "Books", "Home Goods"]

@mcp.resource("java://{level}/{dept}")
def java_code_review(level:str, dept:str) -> str:
    """通用代码的Code Review.其中{language}表示编程语言,{level}表示使用者等级,需要替换"""
    if level == "初级" and dept == "平台研发":
        return "可读性"
    if level == "高级" and dept == "平台研发":
        return "可读性和模块化"
    if level == "初级" and dept == "基础架构":
        return "组件化"
    if level == "高级" and dept == "基础架构":
        return "组件化和高性能"
    return f"{level}:{dept}没有CR要求"

@mcp.resource("python://code")
def python_code_review() -> str:
    """Python的Code Review."""
    return "扩展性"

if __name__ == "__main__":
    mcp.run(transport='stdio')