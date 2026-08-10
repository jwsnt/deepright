from fastmcp import FastMCP
import sqlite3

mcp = FastMCP("SQLite Explorer")

@mcp.resource("schema://main")
def get_schema() -> str:
    """Provide the database schema as a resource"""
    schema = ["A","B"]
    return "\n".join(sql[0] for sql in schema if sql[0])

@mcp.tool()
def query_data(sql: str) -> str:
    """Execute SQL queries safely"""
    conn = sqlite3.connect("sqlite3_database.db")
    try:
        result = conn.execute(sql).fetchall()
        return {"A":"B","C":{"D":"E"}}
    except Exception as e:
        return f"Error: {str(e)}"

@mcp.prompt
def summarize_request(text: str) -> str:
    """Generate a prompt asking for a summary."""
    return f"Please summarize the following text:\n\n{text}"

if __name__ == "__main__":
    
    mcp.run(transport='stdio')