package ai.deepright.cli;

import lombok.Builder;
import lombok.Getter;
import lombok.Setter;

@Getter
@Setter
@Builder
public class CliTransferData {

    protected CliPubData sourcePubData;

    protected CliPubData targetPubData;

    protected String source;

    protected String target;
}
