package ai.open.right.workflow.flow.llm.provider.openai;

import ai.open.right.workflow.flow.llm.provider.ProviderFunCallRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.Assert;

import java.util.List;
import java.util.Map;

public class OpenAiStreamFunCall extends OpenAiStream {

    public OpenAiStreamFunCall(ProviderStreamConfig<OpenAiRequest> providerRequestConfig) throws Exception {
        super(providerRequestConfig);
    }

    @Override
    protected void addFunRequest(Map<String, Object> message, List<Map<String, Object>> funCalls) throws Exception {
        if (CollectionUtils.isEmpty(funCalls)) {
            return;
        }
        for (Map<String, Object> funCall : funCalls) {
            int index = MapUtils.getIntValue(funCall, "index");
            Map<String, Object> function = MapUtils.getMap(funCall, "function");
            Assert.notEmpty(function, "Function can not be empty");
            Object args = MapUtils.getObject(function, "arguments");
            String name = MapUtils.getString(function, "name");
            String id = MapUtils.getString(funCall, "id");
            if (!CollectionUtils.isEmpty(this.providerFunRequests) && index < this.providerFunRequests.size()) {
                // Update
                this.updateFunRequest(this.providerFunRequests.get(index), message, funCall, args, name, id);
            } else {
                // Create
                this.createFunRequest(message, funCall, args, name, id);
            }
        }
    }

    protected void afterCreateFunRequest(ProviderFunCallRequest providerFunCallRequest, Map<String, Object> message, Map<String, Object> funCall, Object args, String name, String id) throws Exception {
        if (!StringUtils.isEmpty(this.reasoning)) {
            providerFunCallRequest.setReason(this.reasoning.toString());
        }
    }

    protected void afterUpdateFunRequest(ProviderFunCallRequest providerFunCallRequest, Map<String, Object> message, Map<String, Object> funCall, Object args, String name, String id) throws Exception {
        // 强制覆盖
        if (!StringUtils.isEmpty(this.reasoning)) {
            providerFunCallRequest.setReason(this.reasoning.toString());
        }
    }

    protected void updateFunRequest(ProviderFunCallRequest providerFunCallRequest, Map<String, Object> message, Map<String, Object> funCall, Object args, String name, String id) throws Exception {
        providerFunCallRequest.setNameIfAbsent(name);
        providerFunCallRequest.setIdIfAbsent(id);
        providerFunCallRequest.appendArgs(args);
        Map<String, Object> function = MapUtils.getMap(funCall, "function");
        function.put("arguments", providerFunCallRequest.getArgs());
        function.put("name", providerFunCallRequest.getName());
        Map<String, Object> refer = Map.class.cast(providerFunCallRequest.getRefer());
        refer.putIfAbsent("index", funCall.get("index"));
        refer.putIfAbsent("type", funCall.get("type"));
        refer.putIfAbsent("id", funCall.get("id"));
        refer.put("function", function);
        providerFunCallRequest.setRefer(refer);
        // 刷新Reasoning
        providerFunCallRequest.setReason(!StringUtils.isEmpty(this.reasoning) ? this.reasoning.toString() : "");
        this.afterUpdateFunRequest(providerFunCallRequest, message, funCall, args, name, id);
    }

    protected void createFunRequest(Map<String, Object> message, Map<String, Object> funCall, Object args, String name, String id) throws Exception {
        Assert.hasText(name, "Function's name can not be empty");
        Assert.hasText(id, "Function's id can not be empty");
        ProviderFunCallRequest providerFunCallRequest = ProviderFunCallRequest.builder()
                .reason(!StringUtils.isEmpty(this.reasoning) ? this.reasoning.toString() : "")
                .model(this.request.getModel())
                .api(this.request.getApi())
                .refer(funCall)
                .name(name)
                .args(args)
                .id(id)
                .build();
        this.afterCreateFunRequest(providerFunCallRequest, message, funCall, args, name, id);
        this.addFunRequest(providerFunCallRequest);
    }
}
