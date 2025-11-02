import './StatusBar.css';

function StatusBar({ status }) {
  return (
    <div className="status-bar">
      <span className="status-text">{status.text}</span>
      <span className="status-details">{status.details}</span>
    </div>
  );
}

export default StatusBar;

