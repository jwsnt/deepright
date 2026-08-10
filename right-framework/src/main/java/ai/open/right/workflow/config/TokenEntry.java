package ai.open.right.workflow.config;

import lombok.Builder;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;

@Setter
@Getter
@Builder
@ToString
public class TokenEntry {

    protected String workflow;

    protected String biz;
}
