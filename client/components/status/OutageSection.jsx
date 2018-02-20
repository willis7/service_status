import React, {Component} from 'react';
import PropTypes from 'prop-types';
import OutageList from './OutageList.jsx';

class OutageSection extends Component {
    render(){
        return(
            <div>
                <OutageList {...this.props} />
            </div>
        )
    }
}

OutageSection.propTypes = {
    outages: PropTypes.array.isRequired
}

export default OutageSection
