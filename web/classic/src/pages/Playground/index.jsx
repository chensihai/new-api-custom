import { PlaygroundQueryProvider } from '../../adapters/playground/query-provider';
import { Playground } from '../../features/playground';

function PlaygroundPage() {
  return (
    <PlaygroundQueryProvider>
      <Playground />
    </PlaygroundQueryProvider>
  );
}

export default PlaygroundPage;
