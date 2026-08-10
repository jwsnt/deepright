package ai.open.right.workflow.flow.llm.token;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Getter;
import lombok.NoArgsConstructor;

@Builder
@Getter
@AllArgsConstructor
@NoArgsConstructor
public class TokenData {

    @Builder.Default
    protected Integer thinking = 0;

    // 输入
    @Builder.Default
    protected Integer input = 0;

    @Builder.Default
    protected Integer total = 0;

    @Builder.Default
    protected Integer cache = 0;

    public Integer getThinking() {
        return this.thinking = this.thinking != null ? this.thinking : 0;
    }

    public Integer getInput() {
        return this.input = this.input != null ? this.input : 0;
    }

    public Integer getTotal() {
        return this.total = this.total != null ? this.total : 0;
    }

    public Integer getCache() {
        return this.cache = this.cache != null ? this.cache : 0;
    }

    public Boolean hasData() {
        return total != null && total > 0;
    }
}
