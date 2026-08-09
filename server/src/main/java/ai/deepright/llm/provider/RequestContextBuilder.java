package ai.deepright.llm.provider;

import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.store.history.History;

public class RequestContextBuilder {

    public static final Integer NEXT = 1;

    public static final Integer MAX = 0;

    public static History buildContext(ProviderRequest request, String content, Integer role, Long time) throws Exception {
        History history = new History(request.getMessage());
        history.setModel(request.getModel());
        history.setApi(request.getApi());
        history.setContent(content);
        history.setCreated(time);
        history.setRole(role);
        return history;
    }

    public static History buildContext(ProviderRequest request, String content, Long time) throws Exception {
        return RequestContextBuilder.buildContext(request, content, History.ROLE_USER, time);
    }

    public static History buildContext(ProviderRequest request, String content) throws Exception {
        return RequestContextBuilder.buildContext(request, content, History.ROLE_USER, System.currentTimeMillis());
    }
}
