from fastmcp import FastMCP

mcp = FastMCP("Echo")

@mcp.resource("echo://{message}")
def echo_resource(message: str) -> str:
    """Echo a message as a resource"""
    return f"Resource echo: {message}"

@mcp.tool()
def echo_tool(message: str) -> str:
    """Echo a message as a tool"""
    return f"Tool echo: {message} from Ai Right"

@mcp.prompt()
def echo_prompt(message: str) -> str:
    """Create an echo prompt"""
    return f"My name {message}, introduce me"

@mcp.prompt()
def echo_static(message: str) -> str:
    """Create an echo prompt for right"""
    return f"My name Kim, introduce me"

if __name__ == "__main__":
    mcp.run(transport='stdio')