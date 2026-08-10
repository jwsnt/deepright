package ai.open.right.workflow.flow.llm;

import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.signal.SignalStream;

public interface LLMQueryService {

    public static final String LLM_VOLCENGINE = "volcengine";

    public static final String LLM_ANTHROPIC = "anthropic";

    public static final String LLM_BIGMODEL = "bigmodel";

    public static final String LLM_SEEDREAM = "seedream";

    public static final String LLM_DEEPSEEK = "deepseek";

    public static final String LLM_MINIMAX = "minimax";

    // 特殊用途
    public static final String LLM_UNKNOW = "unknown";

    public static final String LLM_OPENAI = "openai";

    public static final String LLM_XIAOMI = "xiaomi";

    public static final String LLM_VERTEX = "vertex";

    public static final String LLM_GEMINI = "gemini";

    public static final String LLM_COZE = "coze";

    public static final String LLM_KIMI = "kimi";

    public static final String LLM_QWEN = "qwen";

    public void query(LLMQuery llmQuery, LLMConfig llmConfig, SignalStream signalStream) throws Exception;

    public void query(LLMQuery llmQuery, LLMConfig llmConfig) throws Exception;
}
