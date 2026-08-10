package ai.open.right.workflow.config;

// 用于Fun Call方法编码
public interface NamesService {

    // 编码思考链（Workflow）前缀，用于反向解析
    public final static String PREFIX_WORKFLOW = "Workflow_";

    // 编码MCP Resource前缀，用于反向解析
    public final static String PREFIX_RESOURCE = "Resource_";

    // 编码MCP Prompt前缀，用于反向解析
    public final static String PREFIX_PROMPT = "Prompt_";

    // 编码MCP Tools前缀，用于反向解析
    public final static String PREFIX_TOOLS = "Tools_";

    // 使用指定前缀，客户端，方法名编码
    public String encode(String prefix, String client, String name) throws Exception;

    // 编码名解码为客户端+方法
    public String[] decode(String name) throws Exception;

    public Boolean isPrefixWorkflow(String name) throws Exception;

    public Boolean isPrefixResource(String name) throws Exception;

    public Boolean isPrefixPrompt(String name) throws Exception;

    public Boolean isPrefixTools(String name) throws Exception;

    public Boolean isPrefix(String name) throws Exception;
}
