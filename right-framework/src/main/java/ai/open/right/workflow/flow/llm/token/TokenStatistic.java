package ai.open.right.workflow.flow.llm.token;

import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.store.Dimension;

import java.util.List;
import java.util.Set;

public interface TokenStatistic {

    public void stat(ProviderRequest providerRequest, TokenData tokenData) throws Exception;

    public List<TokenData> readAll(Dimension dimension, List<String> model) throws Exception;

    // 无序返回
    public List<TokenData> readAll(Dimension dimension) throws Exception;

    public TokenData read(Dimension dimension, String model) throws Exception;

    public TokenData read(Dimension dimension) throws Exception;

    // 所有处理过的Models（内存态）
    public Set<String> models() throws Exception;
}
