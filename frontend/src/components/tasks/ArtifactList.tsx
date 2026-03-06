import type { Artifact } from '../../types';

interface ArtifactListProps {
  artifacts: Artifact[];
}

export function ArtifactList({ artifacts }: ArtifactListProps) {
  if (!artifacts || artifacts.length === 0) {
    return (
      <div className="bg-gray-800 border border-gray-700 rounded-lg p-6 text-center text-gray-400">
        No artifacts uploaded yet
      </div>
    );
  }

  return (
    <div className="bg-gray-800 border border-gray-700 rounded-lg p-6">
      <h3 className="text-lg font-semibold text-gray-100 mb-4">Artifacts</h3>
      <div className="space-y-3">
        {artifacts.map((artifact) => (
          <div
            key={artifact.id}
            className="flex items-center justify-between p-4 bg-gray-900 rounded-md border border-gray-700"
          >
            <div className="flex-1">
              <div className="flex items-center gap-2 mb-1">
                <span className="text-gray-200 font-medium">
                  {artifact.kind === 'file' && '📄 File'}
                  {artifact.kind === 'url' && '🔗 URL'}
                  {artifact.kind === 'text' && '📝 Text'}
                </span>
                <span className="text-xs px-2 py-1 bg-gray-700 text-gray-300 rounded">
                  {artifact.context}
                </span>
              </div>
              
              {artifact.url && (
                <a
                  href={artifact.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-sm text-blue-400 hover:text-blue-300 break-all"
                >
                  {artifact.url}
                </a>
              )}
              
              {artifact.text_body && (
                <p className="text-sm text-gray-300 mt-1 line-clamp-2">{artifact.text_body}</p>
              )}
              
              <div className="text-xs text-gray-500 mt-1">
                {artifact.mime && <span>Type: {artifact.mime} • </span>}
                {artifact.bytes && <span>Size: {(artifact.bytes / 1024).toFixed(1)} KB • </span>}
                Uploaded {new Date(artifact.created_at).toLocaleDateString()}
              </div>
            </div>
            
            <div className="ml-4">
              {artifact.av_scan_status === 'clean' && (
                <span className="text-green-400 text-sm">✓ Scan OK</span>
              )}
              {artifact.av_scan_status === 'pending' && (
                <span className="text-yellow-400 text-sm">⏳ Scanning</span>
              )}
              {artifact.av_scan_status === 'infected' && (
                <span className="text-red-400 text-sm">⚠️ Infected</span>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
