import type { StatusState } from '../types';

import './StatusBar.css';

interface StatusBarProps {
  status: StatusState;
}

const StatusBar = ({ status }: StatusBarProps): JSX.Element => (
  <div className="status-bar">
    <span className="status-text">{status.text}</span>
    <span className="status-details">{status.details}</span>
  </div>
);

export default StatusBar;

