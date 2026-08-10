from fastmcp import FastMCP
from fastmcp.prompts.prompt import Message

mcp = FastMCP("MultiRolesPrompt")
@mcp.prompt
def debug_session_start(error_message: str) -> list[Message]:
    """Initiates a debugging help session."""
    return [
        Message(role = "user", content = f"your name is:\n{error_message}"),
        Message(role = "assistant", content = "Okay, I can help with that. Can you provide the full traceback and tell me what you were trying to do?")
    ]

if __name__ == "__main__":
    mcp.run(transport='stdio')