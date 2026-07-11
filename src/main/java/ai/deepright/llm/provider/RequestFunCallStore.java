package ai.deepright.llm.provider;

import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.config.NamesService;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallResponse;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.llm.store.history.HistoryPair;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;

import java.util.List;
import java.util.Map;

@Slf4j
// 存储策略
public class RequestFunCallStore {

    public static final String SOURCE = "cli@sub";

    public static void shouldStoreFunCallData(ProviderRequest request, ProviderFunCallRequest funCallRequest, ProviderFunCallResponse funCallResponse, HistoryStore historyStore, NamesService namesService) throws Exception {
        if (StringUtils.equalsIgnoreCase(RequestFunCallStore.SOURCE, SplitUtils.join(namesService.decode(funCallRequest.getName())))) {
            // 去掉无法解析的Reason
            String r = !StringUtils.equalsIgnoreCase(ProviderRequest.REQUEST_GOOGLE, request.getApi()) ? funCallRequest.getReason() : "";
            Map<String, Object> subData = JsonUtils.transfer(funCallRequest.getArgs(), Map.class);
            String w = StringUtils.defaultIfEmpty(MapUtils.getString(subData, "why_do_this"), "");
            // R和W任一不为空
            if (MapUtils.getBoolean(subData, "is_key_step", true) && !StringUtils.isEmpty(r) || !StringUtils.isEmpty(w)) {
                History history = new History(request.getMessage());
                history.setContent(new StringBuffer().append(w).append(System.lineSeparator()).append(r).toString());
                history.setCreated(funCallRequest.getCreated());
                history.setSource(RequestFunCallStore.SOURCE);
                history.setFunction(History.FUN_CHAT);
                history.setModel(request.getModel());
                history.setApi(request.getApi());
                history.setAssistant();
                history.setAnswer();
                historyStore.store(request.getMessage(), request.getRepositories(), List.of(new HistoryPair(history)), request.getExpired(), request.getHistories());
            }
        }
    }
}
