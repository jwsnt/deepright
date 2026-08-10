package ai.open.right.workflow.mcp.client.stdio;

import ai.open.right.WorkflowException;
import ai.open.right.workflow.mcp.McpMethod;
import ai.open.right.workflow.mcp.client.McpClient;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import com.google.common.cache.Cache;
import com.google.common.cache.CacheBuilder;
import com.google.common.collect.ImmutableList;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.ArrayUtils;
import org.springframework.util.CollectionUtils;

import java.io.*;
import java.util.List;
import java.util.Map;
import java.util.concurrent.Callable;

@Slf4j
public class McpStdioClient extends McpClient {

    protected final Cache<String, List<Map<String, Object>>> cache = CacheBuilder.newBuilder().maximumSize(Integer.MAX_VALUE).build();

    protected final Process process;

    public McpStdioClient(String name, Map<String, String> env, String cmd, String... args) throws Exception {
        String[] command = ImmutableList.<String>builder().add(cmd).addAll(ImmutableList.copyOf(ArrayUtils.isEmpty(args) ? new String[]{} : args)).build().toArray(new String[0]);
        try {
            ProcessBuilder processBuilder = new ProcessBuilder(command);
            if (!CollectionUtils.isEmpty(env)) {
                processBuilder.environment().putAll(env);
            }
            // 合并错误流，防止假死
            processBuilder.redirectErrorStream(true);
            this.process = processBuilder.start();
            // 仅同步读取stdout或stdError之一，由Stdio本身保证不双写通道（写入大量数据会导致管道缓冲区填满，子进程阻塞，父进程死锁）
            super.stdOutput = new McpBufferedWriter(new BufferedWriter(new OutputStreamWriter(this.process.getOutputStream())));
            super.stdInput = new McpBufferedReader(new BufferedReader(new InputStreamReader(this.process.getInputStream())));
            this.init(name);
        } catch (Exception e) {
            WorkflowException.dolog(e);
            this.close();
            throw e;
        }
    }

    public McpStdioClient(String name, String cmd, String... args) throws Exception {
        this(name, null, cmd, args);
    }

    public List<Map<String, Object>> resourcesTemplatesList(McpDimension dimension) throws Exception {
        return this.cache.get(McpMethod.KEY_RESOURCES_TEMPLATES_LIST, new ResourceTemplateListCallable());
    }

    public List<Map<String, Object>> resourcesList(McpDimension dimension) throws Exception {
        return this.cache.get(McpMethod.KEY_RESOURCES_LIST, new ResourceListCallable());
    }

    @Override
    public List<Map<String, Object>> promptList(McpDimension dimension) throws Exception {
        return this.cache.get(McpMethod.KEY_PROMPTS_LIST, new PromptListCallable());
    }

    @Override
    public List<Map<String, Object>> toolsList(McpDimension dimension) throws Exception {
        return this.cache.get(McpMethod.KEY_TOOLS_LIST, new ToolsListCallable());
    }

    @Override
    public void close() throws IOException {
        if (this.process != null) {
            // 先销毁子进程，避免僵尸进程
            this.process.destroyForcibly();
        }
        this.cache.cleanUp();
        // 销毁IO
        super.close();
    }

    private class ResourceTemplateListCallable implements Callable<List<Map<String, Object>>> {

        @Override
        public List<Map<String, Object>> call() throws Exception {
            // 缓存，不支持传递McpDimension
            return McpStdioClient.super.resourcesTemplatesList(null);
        }
    }

    private class ResourceListCallable implements Callable<List<Map<String, Object>>> {

        @Override
        public List<Map<String, Object>> call() throws Exception {
            // 缓存，不支持传递McpDimension
            return McpStdioClient.super.resourcesList(null);
        }
    }


    private class PromptListCallable implements Callable<List<Map<String, Object>>> {

        @Override
        public List<Map<String, Object>> call() throws Exception {
            // 缓存，不支持传递McpDimension
            return McpStdioClient.super.promptList(null);
        }
    }

    private class ToolsListCallable implements Callable<List<Map<String, Object>>> {

        @Override
        public List<Map<String, Object>> call() throws Exception {
            // 缓存，不支持传递McpDimension
            return McpStdioClient.super.toolsList(null);
        }
    }
}
