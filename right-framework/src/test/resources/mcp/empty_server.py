from fastmcp import FastMCP
import sqlite3

mcp = FastMCP("Empty Server")

@mcp.resource("schema://main")
def get_schema() -> str:
    """Provide the database schema as a resource"""
    schema = ["A","B"]
    return "\n".join(sql[0] for sql in schema if sql[0])

@mcp.prompt()
def analyze_table(table: str) -> str:
    """Create a prompt template for analyzing tables"""
    return f"""Please analyze this database table:
Table: {table}
Schema: 
{get_schema()}

What insights can you provide about the structure and relationships?"""
if __name__ == "__main__":
    
    mcp.run(transport='stdio')